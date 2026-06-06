// Package pip registers the lowfat "pip pip3" output filter (pure-DSL).
package pip

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "pip-compact",
	Version:     "0.6.8",
	Description: "Compact pip install output for LLM contexts",
	Commands:    []string{"pip", "pip3"},
	Subcommands: []string{"install"},
}

// New builds the pip pip3 filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
