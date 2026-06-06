package lf

import (
	"fmt"
	"regexp"
	"strings"
)

// ShellRunner executes a filter's shell: / or-shell: ops. The bundled filters'
// awk/sed snippets are reimplemented in pure Go behind this interface, so the
// engine never spawns a subprocess. Pure-DSL filters pass a nil ShellRunner.
type ShellRunner interface {
	// RunShell runs cmd (with $1..$9 macro args already expanded) against stdin
	// and returns its output. ctx carries level/sub/exit/args for ops that need
	// them (e.g. compact-diff keys off level).
	RunShell(cmd string, stdin string, ctx *ExecCtx) (string, error)
}

// ExecCtx is the per-invocation context for the executor.
type ExecCtx struct {
	Sub      string
	Level    Level
	ExitCode int
	Args     []string
	Shell    ShellRunner // may be nil for pure-DSL filters
}

// Execute runs the matching rule against input and returns filtered output.
// No rule match => passthrough. Non-empty output always ends in a newline.
func Execute(rs *RuleSet, ctx *ExecCtx, input string) (string, error) {
	rule := rs.selectRule(ctx.Sub, ctx.Level)
	if rule == nil {
		return input, nil
	}
	out, err := runOps(rule.Ops, ctx, input, rs, nil)
	if err != nil {
		return "", err
	}
	return ensureTrailingNewline(out), nil
}

func ensureTrailingNewline(s string) string {
	if s != "" && !strings.HasSuffix(s, "\n") {
		return s + "\n"
	}
	return s
}

func runOps(ops []Op, ctx *ExecCtx, input string, rs *RuleSet, macroArgs []MacroArg) (string, error) {
	state := input
	for i := range ops {
		s, err := applyOp(&ops[i], state, input, ctx, rs, macroArgs)
		if err != nil {
			return "", err
		}
		state = s
	}
	return state, nil
}

func applyOp(op *Op, state, raw string, ctx *ExecCtx, rs *RuleSet, macroArgs []MacroArg) (string, error) {
	switch op.Kind {
	case OpKeep:
		return filterLines(state, func(l string) bool { return op.Pattern.MatchString(l) }), nil
	case OpDrop:
		return filterLines(state, func(l string) bool { return !op.Pattern.MatchString(l) }), nil
	case OpHead:
		return takeHead(state, resolveHead(op, ctx.Level)), nil
	case OpTail:
		return takeTail(state, resolveHead(op, ctx.Level)), nil
	case OpOr:
		if strings.TrimSpace(state) == "" {
			return op.OrText, nil
		}
		return state, nil
	case OpOrShell:
		if strings.TrimSpace(state) == "" {
			return runShell(op.Body, raw, ctx, macroArgs)
		}
		return state, nil
	case OpRaw:
		return state, nil
	case OpCascade:
		for i := range op.Branches {
			br := &op.Branches[i]
			if br.Guard == nil || guardMatches(br.Guard, ctx) {
				return runOps(br.Ops, ctx, state, rs, macroArgs)
			}
		}
		return state, nil
	case OpShell:
		return runShell(op.Body, state, ctx, macroArgs)
	case OpPython:
		return "", fmt.Errorf("python: ops are not supported by the pure-Go engine")
	case OpMacroCall:
		def := rs.findDefine(op.MacroName)
		if def == nil {
			return "", fmt.Errorf("undefined macro `%s`", op.MacroName)
		}
		if len(op.MacroArgs) != len(def.Params) {
			return "", fmt.Errorf("macro `%s` expects %d arg(s), got %d", op.MacroName, len(def.Params), len(op.MacroArgs))
		}
		return runOps(def.Ops, ctx, state, rs, op.MacroArgs)
	case OpSplit:
		a, b := splitAtFirstMatch(state, op.Pattern)
		preOut := a
		if len(op.Pre) > 0 {
			var err error
			if preOut, err = runOps(op.Pre, ctx, a, rs, macroArgs); err != nil {
				return "", err
			}
		}
		postOut := b
		if len(op.Post) > 0 {
			var err error
			if postOut, err = runOps(op.Post, ctx, b, rs, macroArgs); err != nil {
				return "", err
			}
		}
		return joinNonempty(preOut, postOut), nil
	}
	return "", fmt.Errorf("unhandled op kind %d", op.Kind)
}

// runShell expands $1..$9 macro args then delegates to the filter's ShellRunner.
func runShell(body, stdin string, ctx *ExecCtx, macroArgs []MacroArg) (string, error) {
	if ctx.Shell == nil {
		return "", fmt.Errorf("shell op encountered but no ShellRunner is configured for this filter")
	}
	return ctx.Shell.RunShell(expandArgs(body, macroArgs), stdin, ctx)
}

func guardMatches(g *Guard, ctx *ExecCtx) bool {
	for i := range g.Atoms {
		if !atomMatches(&g.Atoms[i], ctx) {
			return false
		}
	}
	return true
}

func atomMatches(a *Atom, ctx *ExecCtx) bool {
	switch a.Kind {
	case AtomExit:
		if a.Exit == ExitOk {
			return ctx.ExitCode == 0
		}
		return ctx.ExitCode != 0
	case AtomLevel:
		return a.Level == ctx.Level
	case AtomFlag:
		return flagMatches(a.Flag, ctx.Args)
	}
	return false
}

// flagMatches mirrors lowfat's flag guard: bare presence (--stat, --output=json),
// or flag+value in `-o yaml` / `-o=yaml` / glued short `-oyaml` forms.
func flagMatches(spec string, args []string) bool {
	idx := strings.IndexFunc(spec, func(r rune) bool { return r == ' ' || r == '\t' })
	if idx < 0 {
		for _, a := range args {
			if a == spec {
				return true
			}
			if name, _, ok := strings.Cut(a, "="); ok && name == spec {
				return true
			}
		}
		return false
	}
	flag := spec[:idx]
	value := strings.TrimSpace(spec[idx+1:])
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	glued := flag + "=" + value
	for _, a := range args {
		if a == glued {
			return true
		}
	}
	if len(flag) == 2 {
		short := flag + value
		for _, a := range args {
			if a == short {
				return true
			}
		}
	}
	return false
}

func resolveHead(op *Op, level Level) int {
	if op.HeadAuto {
		return level.HeadLimit(30)
	}
	return op.HeadN
}

// rustLines mirrors Rust's str::lines(): split on '\n', drop a trailing empty
// produced by a final newline, and strip a trailing '\r' per line.
func rustLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSuffix(parts[i], "\r")
	}
	return parts
}

func filterLines(s string, keep func(string) bool) string {
	var out []string
	for _, l := range rustLines(s) {
		if keep(l) {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func takeHead(s string, n int) string {
	lines := rustLines(s)
	if n < len(lines) {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func takeTail(s string, n int) string {
	lines := rustLines(s)
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	return strings.Join(lines[start:], "\n")
}

func splitAtFirstMatch(s string, re *regexp.Regexp) (string, string) {
	var pre, post strings.Builder
	inPost := false
	for _, l := range rustLines(s) {
		if !inPost && re.MatchString(l) {
			inPost = true
		}
		buf := &pre
		if inPost {
			buf = &post
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(l)
	}
	return pre.String(), post.String()
}

func joinNonempty(a, b string) string {
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "\n" + b
	}
}

// expandArgs replaces $1..$9 with macro positional args. Other $NAME tokens are
// left intact for the ShellRunner to interpret from ctx.
func expandArgs(body string, args []MacroArg) string {
	if len(args) == 0 {
		return body
	}
	var out strings.Builder
	out.Grow(len(body))
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == '$' && i+1 < len(body) {
			n := body[i+1]
			if n >= '1' && n <= '9' {
				if idx := int(n - '0'); idx <= len(args) {
					a := args[idx-1]
					if a.IsNumber {
						out.WriteString(fmt.Sprintf("%d", a.Number))
					} else {
						out.WriteString(a.Str)
					}
					i++
					continue
				}
			}
		}
		out.WriteByte(c)
	}
	return out.String()
}
