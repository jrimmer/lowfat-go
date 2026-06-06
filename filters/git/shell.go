package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jrimmer/lowfat-go/internal/awk"
	"github.com/jrimmer/lowfat-go/lf"
)

// shell implements lf.ShellRunner for the git filter's shell: / or-shell: ops.
// It recognizes the specific awk/sed snippets in git/filter.lf (post macro-arg
// expansion) and dispatches to native Go. This is the pure-Go replacement for
// shelling out to `sh -c`.
type shell struct{}

func (shell) RunShell(cmd, stdin string, ctx *lf.ExecCtx) (string, error) {
	c := strings.TrimSpace(cmd)
	switch {
	// define abbrev-hash: sed -E 's/^commit ([0-9a-f]{12})[0-9a-f]{28}/commit \1/'
	case strings.HasPrefix(c, "sed -E 's/^commit"):
		return abbrevHash(stdin), nil

	// define compact-diff(limit): awk state machine over a unified diff.
	case strings.Contains(c, "in_hunk") && strings.Contains(c, "lim="):
		lim, err := parseLim(c)
		if err != nil {
			return "", err
		}
		return compactDiff(stdin, lim, ctx.Level), nil

	// or-shell fallback: `awk 'NF' | head -50` (drop blank lines, take 50).
	case strings.Contains(c, "awk 'NF'"):
		return awk.Head(awk.KeepLines(stdin, func(l string) bool {
			return strings.TrimSpace(l) != ""
		}), 50), nil
	}
	return "", fmt.Errorf("git filter: unrecognized shell command: %q", firstLine(c))
}

var commitHashRe = regexp.MustCompile(`^commit ([0-9a-f]{12})[0-9a-f]{28}`)

// abbrevHash mirrors `sed -E 's/^commit ([0-9a-f]{12})[0-9a-f]{28}/commit \1/'`:
// truncate a 40-char commit hash to its first 12 chars, per line.
func abbrevHash(input string) string {
	return awk.MapLines(input, func(l string) string {
		return commitHashRe.ReplaceAllString(l, "commit ${1}")
	})
}

var limRe = regexp.MustCompile(`lim=([0-9]+)`)

func parseLim(cmd string) (int, error) {
	m := limRe.FindStringSubmatch(cmd)
	if m == nil {
		return 0, fmt.Errorf("compact-diff: could not parse lim= from command")
	}
	n := 0
	for _, ch := range m[1] {
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

// compactDiff mirrors the git filter's awk:
//
//	BEGIN { in_hunk=0; n=0 }
//	n>=lim { exit }
//	/^diff / { in_hunk=0; print; n++; next }
//	/^@@ /  { in_hunk=1
//	          if (lvl=="ultra" && match($0,/ @@/)) print substr($0,1,RSTART+2)
//	          else print
//	          n++; next }
//	lvl=="ultra" { next }
//	in_hunk && /^[+-]/ { print; n++ }
func compactDiff(input string, lim int, level lf.Level) string {
	var out []string
	inHunk := false
	n := 0
	ultra := level == lf.Ultra
	for _, l := range awk.Lines(input) {
		if n >= lim {
			break
		}
		switch {
		case strings.HasPrefix(l, "diff "):
			inHunk = false
			out = append(out, l)
			n++
		case strings.HasPrefix(l, "@@ "):
			inHunk = true
			if ultra {
				// match($0, / @@/): first occurrence; RSTART is 1-based.
				// substr($0,1,RSTART+2) keeps up to and including " @@".
				if idx := strings.Index(l, " @@"); idx >= 0 {
					out = append(out, l[:idx+3])
				} else {
					out = append(out, l)
				}
			} else {
				out = append(out, l)
			}
			n++
		default:
			if ultra {
				continue
			}
			if inHunk && (strings.HasPrefix(l, "+") || strings.HasPrefix(l, "-")) {
				out = append(out, l)
				n++
			}
		}
	}
	return strings.Join(out, "\n")
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
