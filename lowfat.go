// Package lowfat embeds lowfat's command-output filtering engine as a pure-Go
// library. It reduces the token cost of CLI output (git, docker, ls, …) before
// that output reaches an LLM, with no subprocess execution and no dynamic
// linking — tool filters are registered at compile time.
//
// Basic use:
//
//	import (
//	    "go.rimmer.net/lowfat"
//	    _ "go.rimmer.net/lowfat/filters/all" // register built-in filters
//	)
//
//	res, err := lowfat.Filter("git", []string{"diff"}, output,
//	    lowfat.Options{Level: lowfat.Ultra, ExitCode: 0})
//	// res.Output is the compacted text; res.InputTokens-res.OutputTokens were saved.
//
// To control exactly which filters are available, build your own Registry with
// the per-filter New() constructors instead of importing filters/all.
package lowfat

import (
	"fmt"

	"go.rimmer.net/lowfat/lf"
)

// Level re-exports lf.Level so callers need not import lf.
type Level = lf.Level

const (
	Lite  = lf.Lite
	Full  = lf.Full
	Ultra = lf.Ultra
)

// ParseLevelString parses "lite"/"full"/"ultra" into a Level.
func ParseLevelString(s string) (Level, error) { return lf.ParseLevel(s) }

// Options configures a single Filter call.
type Options struct {
	// Level is the compression intensity. The zero value is Lite; most callers
	// set Full or Ultra. See WithDefaults.
	Level Level
	// ExitCode is the filtered command's exit status. Drives `if exit failed`
	// branches (e.g. grep no-match, find permission errors).
	ExitCode int
	// MinTokens skips filtering when the input is smaller than this many tokens
	// (filtering overhead exceeds any saving). 0 means always filter. lowfat's
	// runtime uses 128.
	MinTokens int
	// levelSet tracks whether Level was explicitly provided, so the default of
	// Full can be applied without colliding with the valid zero value (Lite).
	levelSet bool
}

// FullLevel returns Options with Level set to Full (the conventional default),
// a convenience so the zero value (Lite) is never applied by surprise.
func FullLevel() Options { return Options{Level: Full, levelSet: true} }

// At returns a copy of o with Level set explicitly.
func (o Options) At(l Level) Options { o.Level = l; o.levelSet = true; return o }

// Result reports the outcome of a Filter call.
type Result struct {
	Output       string // filtered text, or the original if no filter ran
	Filtered     bool   // whether a filter actually ran
	FilterName   string // e.g. "git-compact" ("" if none matched)
	InputTokens  int
	OutputTokens int // savings = InputTokens - OutputTokens
}

// ToolFilter is a registered tool filter: a manifest, a parsed .lf ruleset, and
// an optional ShellRunner for that tool's native awk/sed ops. The package-level
// Filter function (not this type) is the usual entry point.
type ToolFilter struct {
	manifest Manifest
	ruleset  *lf.RuleSet
	shell    lf.ShellRunner
}

// Manifest returns a copy of the filter's manifest.
func (f *ToolFilter) Manifest() Manifest { return f.manifest }

// run executes the filter against output for the given derived sub/opts.
func (f *ToolFilter) run(sub string, args []string, output string, opt Options) (string, error) {
	ctx := &lf.ExecCtx{
		Sub:      sub,
		Level:    opt.Level,
		ExitCode: opt.ExitCode,
		Args:     args,
		Shell:    f.shell,
	}
	return lf.Execute(f.ruleset, ctx, output)
}

// Registry maps command names to filters. It is not safe for concurrent
// registration, but concurrent Filter calls after setup are safe (read-only).
type Registry struct {
	byCommand map[string]*ToolFilter
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byCommand: map[string]*ToolFilter{}}
}

// Register adds a filter, claiming each of its manifest Commands. It is an error
// for two filters to claim the same command.
func (r *Registry) Register(f *ToolFilter) error {
	if f == nil {
		return fmt.Errorf("lowfat: cannot register a nil filter")
	}
	if len(f.manifest.Commands) == 0 {
		return fmt.Errorf("lowfat: filter %q declares no commands", f.manifest.Name)
	}
	for _, cmd := range f.manifest.Commands {
		if existing, ok := r.byCommand[cmd]; ok {
			return fmt.Errorf("lowfat: command %q already registered by %q", cmd, existing.manifest.Name)
		}
	}
	for _, cmd := range f.manifest.Commands {
		r.byCommand[cmd] = f
	}
	return nil
}

// MustRegister is Register that panics on error; intended for package init().
func (r *Registry) MustRegister(f *ToolFilter) {
	if err := r.Register(f); err != nil {
		panic(err)
	}
}

// ForCommand returns the filter handling command, if any.
func (r *Registry) ForCommand(command string) (*ToolFilter, bool) {
	f, ok := r.byCommand[command]
	return f, ok
}

// Commands returns the sorted-insertion list of registered command names.
func (r *Registry) Commands() []string {
	out := make([]string, 0, len(r.byCommand))
	for c := range r.byCommand {
		out = append(out, c)
	}
	return out
}

// Filter compacts output for command with the given args (everything after the
// command name; args[0] is the subcommand, mirroring lowfat). If no filter is
// registered for command, output is returned unchanged with Filtered=false.
func (r *Registry) Filter(command string, args []string, output string, opt Options) (Result, error) {
	if !opt.levelSet && opt.Level == Lite {
		opt.Level = Full // apply conventional default without clobbering an explicit Lite
	}
	in := EstimateTokens(output)
	res := Result{Output: output, InputTokens: in, OutputTokens: in}

	f, ok := r.byCommand[command]
	if !ok {
		return res, nil
	}
	res.FilterName = f.manifest.Name

	if opt.MinTokens > 0 && in < opt.MinTokens {
		return res, nil // too small to be worth filtering
	}

	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	out, err := f.run(sub, args, output, opt)
	if err != nil {
		// Degrade to passthrough — never make output worse than no filter.
		return res, err
	}
	res.Output = out
	res.Filtered = true
	res.OutputTokens = EstimateTokens(out)
	return res, nil
}

// ── default registry ───────────────────────────────────────────────

var defaultRegistry = NewRegistry()

// Default returns the process-wide registry that built-in filter packages
// register into via their init() functions.
func Default() *Registry { return defaultRegistry }

// MustRegister registers f into the default registry; for filter package init().
func MustRegister(f *ToolFilter) { defaultRegistry.MustRegister(f) }

// Filter runs against the default registry. Import the filter packages you need
// (or go.rimmer.net/lowfat/filters/all) so they self-register.
func Filter(command string, args []string, output string, opt Options) (Result, error) {
	return defaultRegistry.Filter(command, args, output, opt)
}
