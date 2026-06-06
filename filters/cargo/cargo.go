// Package cargo registers the lowfat "cargo" output filter. Its only shell op is
// an `else-shell: printf 'cargo %s: ok'` status line, reimplemented natively.
package cargo

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "cargo-compact",
	Version:     "0.6.8",
	Description: "Compact cargo build/test/check/clippy/run output for LLM contexts",
	Commands:    []string{"cargo"},
	Subcommands: []string{"build", "test", "check", "clippy", "run", "add", "update"},
}

// New builds the cargo filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, shell{}) }

func init() { lowfat.MustRegister(New()) }
