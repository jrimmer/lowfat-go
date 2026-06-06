// Package makecmd registers the lowfat "make" output filter (pure-DSL).
package makecmd

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "make-compact",
	Version:     "0.6.8",
	Description: "Compact make output for LLM contexts",
	Commands:    []string{"make"},
}

// New builds the make filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
