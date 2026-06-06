// Package fd registers the lowfat "fd" output filter (pure-DSL).
package fd

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "fd-compact",
	Version:     "0.6.8",
	Description: "Compact fd output for LLM contexts",
	Commands:    []string{"fd"},
}

// New builds the fd filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
