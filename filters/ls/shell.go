package ls

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jrimmer/lowfat-go/internal/awk"
	"github.com/jrimmer/lowfat-go/lf"
)

// shell implements lf.ShellRunner for the ls filter's shell: ops.
type shell struct{}

func (shell) RunShell(cmd, stdin string, ctx *lf.ExecCtx) (string, error) {
	c := strings.TrimSpace(cmd)
	switch {
	// *, ultra: awk '{print $NF}'  (keep only the filename column)
	case c == "awk '{print $NF}'":
		return awk.MapLines(stdin, func(l string) string {
			f := awk.Fields(l)
			if len(f) == 0 {
				return ""
			}
			return f[len(f)-1]
		}), nil

	// define compact-long-form: collapse `ls -l` rows to `<type> <size> <name>`.
	case strings.Contains(c, "NF >= 9"):
		return compactLongForm(stdin), nil
	}
	return "", fmt.Errorf("ls filter: unrecognized shell command: %q", firstLine(c))
}

var longFormRe = regexp.MustCompile(`^[-dlbcps][-r]`)

// compactLongForm mirrors:
//
//	awk '
//	  NF >= 9 && $1 ~ /^[-dlbcps][-r]/ {
//	      t = substr($1,1,1); s = $5;
//	      name = $9; for (i=10; i<=NF; i++) name = name " " $i
//	      print t, s, name; next
//	  }
//	  { print }
//	'
func compactLongForm(input string) string {
	return awk.MapLines(input, func(l string) string {
		f := awk.Fields(l)
		if len(f) >= 9 && longFormRe.MatchString(f[0]) {
			t := f[0][:1]
			s := f[4]
			name := strings.Join(f[8:], " ")
			return t + " " + s + " " + name
		}
		return l
	})
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
