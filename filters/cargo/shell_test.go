package cargo

import (
	"strings"
	"testing"

	"github.com/jrimmer/lowfat-go/lf"
)

func TestShellRunShell(t *testing.T) {
	out, err := (shell{}).RunShell(" printf 'cargo %s: ok' \"$sub\" ", "ignored", &lf.ExecCtx{Sub: "test"})
	if err != nil {
		t.Fatalf("RunShell returned error: %v", err)
	}
	if out != "cargo test: ok" {
		t.Fatalf("RunShell output = %q", out)
	}
	_, err = (shell{}).RunShell("awk '{print $1}'", "", &lf.ExecCtx{})
	if err == nil || !strings.Contains(err.Error(), "unrecognized shell command") {
		t.Fatalf("RunShell unrecognized error = %v", err)
	}
}
