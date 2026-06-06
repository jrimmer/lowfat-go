// Package grep registers the lowfat "grep" output filter (pure-DSL; no shell ops).
package grep

import (
	_ "embed"

	"go.rimmer.net/lowfat"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:     "grep-compact",
	Version:  "0.6.8",
	Commands: []string{"grep"},
}

// New builds the grep filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, nil)
}

func init() { lowfat.MustRegister(New()) }
