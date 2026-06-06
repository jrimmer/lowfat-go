// Package rg registers the lowfat "rg" output filter (pure-DSL).
package rg

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "rg-compact",
	Version:     "0.6.8",
	Description: "Compact ripgrep output for LLM contexts",
	Commands:    []string{"rg"},
}

// New builds the rg filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
