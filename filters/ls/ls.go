// Package ls registers the lowfat "ls" output filter, including pure-Go
// reimplementations of its awk shell ops (last-field, long-form collapse).
package ls

import (
	_ "embed"

	"go.rimmer.net/lowfat"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "ls-compact",
	Version:     "0.6.8",
	Description: "Compact ls output for LLM contexts",
	Commands:    []string{"ls"},
}

// New builds the ls filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, shell{})
}

func init() { lowfat.MustRegister(New()) }
