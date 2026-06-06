// Package gotool registers the lowfat "go" output filter (pure-DSL).
package gotool

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "go-compact",
	Version:     "0.6.8",
	Description: "Compact go build/test/vet/mod output for LLM contexts",
	Commands:    []string{"go"},
	Subcommands: []string{"build", "test", "vet", "mod"},
}

// New builds the go filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
