// Package git registers the lowfat "git" output filter, including pure-Go
// reimplementations of its awk/sed shell ops (compact-diff, abbrev-hash).
package git

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "git-compact",
	Version:     "0.6.8",
	Description: "Compact git status/diff/log/show output for LLM contexts",
	Commands:    []string{"git"},
	Subcommands: []string{"status", "diff", "log", "show"},
}

// New builds the git filter for registration into a custom registry.
func New() *lowfat.ToolFilter {
	return lowfat.MustNewFilter(manifest, filterLF, shell{})
}

func init() { lowfat.MustRegister(New()) }
