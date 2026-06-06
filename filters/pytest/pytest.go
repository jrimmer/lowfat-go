// Package pytest registers the lowfat "pytest" output filter (pure-DSL).
package pytest

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "pytest-compact",
	Version:     "0.6.8",
	Description: "Compact pytest output for LLM contexts",
	Commands:    []string{"pytest"},
}

// New builds the pytest filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
