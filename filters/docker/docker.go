// Package docker registers the lowfat "docker" output filter, including pure-Go
// reimplementations of its awk/sed shell ops (ps/images column extraction,
// whitespace collapse).
package docker

import (
	_ "embed"

	"go.rimmer.net/lowfat"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "docker-compact",
	Version:     "0.6.8",
	Description: "Compact docker ps/images/logs/build/pull/compose output for LLM contexts",
	Commands:    []string{"docker"},
	Subcommands: []string{"ps", "images", "logs", "build", "pull", "compose"},
}

// New builds the docker filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, shell{})
}

func init() { lowfat.MustRegister(New()) }
