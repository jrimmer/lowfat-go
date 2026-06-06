package cargo

import (
	"fmt"
	"strings"

	"github.com/jrimmer/lowfat-go/lf"
)

// shell implements lf.ShellRunner for cargo's lone shell op:
//
//	else-shell: printf 'cargo %s: ok' "$sub"
//
// which prints a one-line OK status when a build/check/add/update produced no
// errors. $sub is taken from the exec context.
type shell struct{}

func (shell) RunShell(cmd, stdin string, ctx *lf.ExecCtx) (string, error) {
	c := strings.TrimSpace(cmd)
	if strings.HasPrefix(c, "printf 'cargo %s: ok'") {
		return "cargo " + ctx.Sub + ": ok", nil
	}
	return "", fmt.Errorf("cargo filter: unrecognized shell command: %q", c)
}
