// Package tree registers the lowfat "tree" output filter (pure-DSL; no shell ops).
package tree

import (
	_ "embed"

	"go.rimmer.net/lowfat"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:     "tree-compact",
	Version:  "0.6.8",
	Commands: []string{"tree"},
}

// New builds the tree filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, nil)
}

func init() { lowfat.MustRegister(New()) }
