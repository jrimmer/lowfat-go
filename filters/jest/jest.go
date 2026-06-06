// Package jest registers the lowfat "jest vitest" output filter (pure-DSL).
package jest

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "jest-compact",
	Version:     "0.6.8",
	Description: "Compact jest/vitest output for LLM contexts",
	Commands:    []string{"jest", "vitest"},
}

// New builds the jest vitest filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
