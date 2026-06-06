// Package kubectl registers the lowfat "kubectl" output filter.
//
// Upstream lowfat prunes `kubectl get -o yaml` server-side noise with a PyYAML
// script. This pure-Go port replaces that with a native, dependency-free
// indent-based YAML pruner (see shell.go). Because the native pruner cannot be
// byte-for-byte identical to PyYAML's serializer, the `get -o yaml` path is
// covered by dedicated unit tests rather than the Rust golden oracle; the
// logs/events/raw paths remain pure-DSL and are golden-tested.
package kubectl

import (
	_ "embed"

	"github.com/jrimmer/lowfat-go"
)

//go:embed filter.lf
var filterLF string

var manifest = lowfat.Manifest{
	Name:        "kubectl-compact",
	Version:     "0.6.8",
	Description: "Compact kubectl get/logs/events/describe/top output for LLM contexts",
	Commands:    []string{"kubectl"},
	Subcommands: []string{"get", "logs", "events", "describe", "top"},
}

// New builds the kubectl filter for registration into a custom registry.
func New() *lowfat.ToolFilter { return lowfat.MustNewFilter(manifest, filterLF, shell{}) }

func init() { lowfat.MustRegister(New()) }
