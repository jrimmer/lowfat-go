// Package npm registers the lowfat "npm" output filter (pure-DSL).
package npm

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "npm-compact",
	Version:     "0.6.8",
	Description: "Compact npm install/test/audit/run output for LLM contexts",
	Commands:    []string{"npm"},
	Subcommands: []string{"install", "i", "test", "t", "audit", "run"},
}

// New builds the npm filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
