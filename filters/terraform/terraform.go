// Package terraform registers the lowfat "terraform tofu" output filter (pure-DSL).
package terraform

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "terraform-compact",
	Version:     "0.6.8",
	Description: "Compact terraform plan/apply output for LLM contexts",
	Commands:    []string{"terraform", "tofu"},
	Subcommands: []string{"plan", "apply"},
}

// New builds the terraform tofu filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, nil) }

func init() { lowfat.MustRegister(New()) }
