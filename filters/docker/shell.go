package docker

import (
	"fmt"
	"strings"

	"github.com/jrimmer/lowfat-go/internal/awk"
	"github.com/jrimmer/lowfat-go/lf"
)

// shell implements lf.ShellRunner for the docker filter's shell: ops.
type shell struct{}

func (shell) RunShell(cmd, stdin string, ctx *lf.ExecCtx) (string, error) {
	c := strings.TrimSpace(cmd)
	switch {
	// ps ultra: printf 'NAME STATUS\n'; tail -n +2 | awk '{print $NF, $(NF-2)}'
	case strings.HasPrefix(c, "printf 'NAME STATUS"):
		return columnExtract(stdin, "NAME STATUS", func(f []string) string {
			return relField(f, 0) + " " + relField(f, -2)
		}), nil

	// images ultra: printf 'REPO TAG SIZE\n'; tail -n +2 | awk '{print $1, $2, $(NF-1)}'
	case strings.HasPrefix(c, "printf 'REPO TAG SIZE"):
		return columnExtract(stdin, "REPO TAG SIZE", func(f []string) string {
			return absField(f, 1) + " " + absField(f, 2) + " " + relField(f, -1)
		}), nil

	// ps/images lite & full: sed 's/  */ /g'
	case c == "sed 's/  */ /g'":
		return awk.CollapseSpaces(stdin), nil
	}
	return "", fmt.Errorf("docker filter: unrecognized shell command: %q", firstLine(c))
}

// columnExtract emits header, then maps each body line (after `tail -n +2`,
// i.e. skipping the docker header row) through fn over its whitespace fields.
func columnExtract(stdin, header string, fn func(fields []string) string) string {
	lines := awk.Lines(stdin)
	out := []string{header}
	for _, l := range lines[min(1, len(lines)):] {
		out = append(out, fn(awk.Fields(l)))
	}
	return strings.Join(out, "\n")
}

// absField returns awk's $i (1-based, from the left). Out of range => "".
func absField(f []string, i int) string {
	if i >= 1 && i <= len(f) {
		return f[i-1]
	}
	return ""
}

// relField returns a field relative to NF: off=0 is $NF (last), off=-1 is
// $(NF-1), off=-2 is $(NF-2). Out of range => "".
func relField(f []string, off int) string {
	idx := len(f) - 1 + off // 0-based; off=0 -> last field ($NF)
	if idx >= 0 && idx < len(f) {
		return f[idx]
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
