package lowfat

import (
	"fmt"

	"go.rimmer.net/lowfat/lf"
)

// Manifest describes a tool filter — the Go equivalent of lowfat's lowfat.toml
// [plugin] table.
type Manifest struct {
	Name        string
	Version     string
	Description string
	// Commands this filter intercepts, e.g. []string{"git"}.
	Commands []string
	// Subcommands it knows about (informational; rule matching is driven by the
	// .lf selectors, not this list). e.g. []string{"status","diff","log","show"}.
	Subcommands []string
}

// NewFilter builds a Filter from a manifest, a .lf source (typically go:embed'd),
// and an optional ShellRunner for the filter's native shell: ops. Pass a nil
// ShellRunner for pure-DSL filters. It returns an error if the .lf fails to parse.
func NewFilter(m Manifest, lfSource string, shell lf.ShellRunner) (*ToolFilter, error) {
	if m.Name == "" {
		return nil, fmt.Errorf("lowfat: manifest is missing Name")
	}
	if len(m.Commands) == 0 {
		return nil, fmt.Errorf("lowfat: manifest %q declares no Commands", m.Name)
	}
	rs, err := lf.Parse(lfSource)
	if err != nil {
		return nil, fmt.Errorf("lowfat: parsing %q filter: %w", m.Name, err)
	}
	return &ToolFilter{manifest: m, ruleset: rs, shell: shell}, nil
}

// MustNewFilter is NewFilter that panics on error; intended for package init().
func MustNewFilter(m Manifest, lfSource string, shell lf.ShellRunner) *ToolFilter {
	f, err := NewFilter(m, lfSource, shell)
	if err != nil {
		panic(err)
	}
	return f
}
