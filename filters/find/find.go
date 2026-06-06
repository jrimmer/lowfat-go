// Package find registers the lowfat "find" output filter (pure-DSL; no shell ops).
package find

import (
	_ "embed"

	"go.rimmer.net/lowfat"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:     "find-compact",
	Version:  "0.6.8",
	Commands: []string{"find"},
}

// New builds the find filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, nil)
}

func init() { lowfat.MustRegister(New()) }
