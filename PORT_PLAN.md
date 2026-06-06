# lowfat → Go port plan

Port lowfat's command-output filtering engine from Rust to **pure Go** so it can be
embedded as a library (the motivating consumer is `scout`, but the API is
deliberately generic). The standalone `lowfat` binary, CLI subcommands, SQLite
metrics, and TOML config layer are **out of scope** — we port the *engine* and the
*bundled filters*, exposed through a small integration API.

This plan was validated by a spike (`git` filter, 12 golden cases, byte-for-byte
parity with the Rust reference). See "Spike result" at the bottom.

## Decisions (settled)

| Question | Decision | Why |
|---|---|---|
| Port vs FFI vs WASM vs subprocess | **Native Go port** | No second toolchain in scout's Docker/CI; WASM breaks awk/sed ops; FFI is cross-compile pain; subprocess is the thing we're killing. |
| awk/sed `shell:` ops | **Reimplemented natively** (pure Go) | No subprocess from inside the filter layer — matters for scout's security posture (deterministic, sandboxed). ~100 LOC total, all field/regex manipulation. |
| Extensibility | **Compile-time pluggable** (registry + `init()` registration), no dynamic linking | New tool type = new package + recompile. Mirrors `database/sql` drivers / `image` decoders. |
| `python:` ops | **Unsupported** (error) | No bundled filter uses python; running arbitrary python in-process is a non-goal. |
| Regex engine | Go `regexp` (RE2) | Rust's `regex` crate is also RE2 — patterns port verbatim, including POSIX classes. |

## Architecture

```
                 ┌─────────────────────────────────────────────┐
   caller ─────▶ │  lowfat.Filter(cmd, args, output, Options)   │  ← integration API
 (e.g. scout)    │      └─ Registry: cmd → *Filter              │
                 └───────────────────┬─────────────────────────┘
                                     │ derive sub = args[0]; build ExecCtx
                                     ▼
                 ┌─────────────────────────────────────────────┐
                 │  lf engine:  Execute(ruleset, ctx, input)    │  ← the DSL (parser + executor)
                 │   keep/drop/head/tail/or/split/match/if/…    │
                 └───────────────────┬─────────────────────────┘
                                     │ shell: / or-shell: ops
                                     ▼
                 ┌─────────────────────────────────────────────┐
                 │  ShellRunner (per filter)                    │  ← native awk/sed, pure Go
                 │   git: compact-diff, abbrev-hash, …          │     (nil for pure-DSL filters)
                 └─────────────────────────────────────────────┘
```

### Module layout

```
github.com/jrimmer/lowfat-go                 (module path — change to taste)
├── lowfat.go            Registry, Filter, Options, Result, package-level Filter()
├── manifest.go          Manifest (name/commands/subcommands) + NewFilter builder
├── tokens.go            EstimateTokens (len+3)/4, lowfat's savings metric
├── lf/                  the .lf DSL engine (parser + executor + Level)
│   ├── level.go  types.go  parse.go  exec.go  doc.go
│   └── *_test.go
├── internal/awk/        reusable native primitives (Fields, CollapseSpaces, …)
├── filters/
│   ├── git/   (filter.lf + git.go + shell.go)   ← has native shell ops
│   ├── docker/(filter.lf + docker.go + shell.go)← has native shell ops
│   ├── ls/    (filter.lf + ls.go + shell.go)    ← has native shell ops
│   ├── tree/  (filter.lf + tree.go)             ← pure-DSL
│   ├── grep/  (filter.lf + grep.go)             ← pure-DSL
│   ├── find/  (filter.lf + find.go)             ← pure-DSL
│   └── all/   (all.go: blank-imports the six → registers them)
├── testdata/            cases.tsv + per-filter samples/ + golden/
├── golden_test.go       differential/golden harness over cases.tsv
├── example_test.go      runnable usage doc (scout-shaped)
└── scripts/regen-golden.sh
```

### The pluggable model (compile-time, no dynamic linking)

A "tool type" is a `*lowfat.Filter` built from three things:

1. a `Manifest` (command names + declared subcommands),
2. an embedded `.lf` ruleset (`//go:embed filter.lf`), and
3. an optional `lf.ShellRunner` providing native Go for that filter's awk/sed ops.

Each filter package registers itself into the **default registry** in `init()`, and
also exposes `New()` for explicit registration into a custom registry:

```go
// filters/git/git.go
func New() *lowfat.Filter { /* parse embedded .lf, attach ShellRunner */ }
func init() { lowfat.MustRegister(New()) }
```

Adding a new tool (say `kubectl`):
1. `mkdir filters/kubectl`, add `filter.lf`, `kubectl.go` (+ `shell.go` if it has awk/sed),
2. add `_ ".../filters/kubectl"` to `filters/all/all.go`,
3. recompile.

No reflection, no plugins, no `.so` loading — just Go packages wired at build time.

### Integration API (generic; verified against scout's needs)

```go
type Options struct {
    Level     lowfat.Level // Lite | Full | Ultra (default Full)
    ExitCode  int          // command's exit status (drives `if exit failed`)
    MinTokens int          // skip filtering below this many tokens (lowfat uses 128; 0 = always)
}

type Result struct {
    Output       string // filtered (or original, if no filter / skipped)
    Filtered     bool   // did a filter actually run?
    FilterName   string // e.g. "git-compact"
    InputTokens  int
    OutputTokens int    // savings = InputTokens - OutputTokens
}

func Filter(command string, args []string, output string, opt Options) (Result, error)
func (r *Registry) Filter(command string, args []string, output string, opt Options) (Result, error)
```

Scout usage (it owns the sandbox, so it already has argv + combined output + exit code):

```go
res, _ := lowfat.Filter("git", []string{"diff"}, toolOutput,
    lowfat.Options{Level: lowfat.Ultra, ExitCode: code})
sendToLLM(res.Output) // res.InputTokens-res.OutputTokens tokens saved
```

This is not scout-specific: any agent/harness with (command, args, output) can call it.
Subcommand derivation matches lowfat exactly: `sub = args[0]` (the token after the
command); `ctx.args = args` (so `if --stat`-style flag guards see the full arg list).

## Test strategy (comprehensive)

1. **Golden parity (the oracle is the Rust binary).** `testdata/cases.tsv` enumerates
   `(filter, sample, sub, level, exit) → golden`. `scripts/regen-golden.sh` produces
   each golden by piping the sample through the **real Rust `lowfat filter`**. The Go
   `golden_test.go` reads the same TSV, runs the Go registry, and asserts byte-for-byte
   equality. Goldens are committed, so the test needs no cargo at run time.
   Coverage: all 6 filters × {ultra,lite,full}, plus `exit!=0` cases for the
   `if exit failed` branches (grep/find/tree) and empty-output `or "…"` fallbacks.
2. **Engine unit tests** (`lf/*_test.go`): parser constructs (define/macro, match, if/elif/else,
   split pre/post, inline vs indented), executor ops (keep/drop/head/tail/or/split, glob
   sub-matching, level head-limit scaling, flag guards), and `.lines()` edge cases.
3. **Native op tests** (per filter `shell_test.go`): each awk/sed reimplementation against
   crafted inputs incl. degenerate rows.
4. **Negative controls** baked into the harness run (proven in the spike): a corrupted
   golden must fail; disabling a native op must fail — guards against a vacuous suite.

## Phases

1. **Engine** — promote the spike's `lf` package; refactor `shell:` execution behind a
   `ShellRunner` interface carried on `ExecCtx`. Add engine unit tests. ✅ design proven
2. **Framework** — `Manifest`, `Filter`, `Registry`, `Options/Result`, `tokens.go`.
3. **Filters** — six packages; native `ShellRunner`s for git/docker/ls; pure-DSL for
   tree/grep/find; `filters/all` aggregator.
4. **Harness** — `cases.tsv`, synthetic samples for uncovered subs/exit paths,
   `regen-golden.sh`, `golden_test.go`, `example_test.go`.
5. **Verify** — `go test ./...` green; differential check vs Rust; `go vet`.

## Non-goals (explicit)

- `.lowfat` TOML config resolution, env-var precedence, conditional pipelines from config
  (the `.lf` files carry their own level/exit conditionals).
- SQLite metrics persistence, `stats`/`history` (we return per-call token counts instead).
- CLI surface: `hook`, `shell-init`, `opencode`, `plugin` scaffolding, `rewrite`.
- `python:` ops and arbitrary user-authored `shell:`/`python:` plugins (curated filters only).
- The outer `proc_normalize` pass (the CLI `lowfat filter` path — our parity target —
  does not apply it; the `lowfat run` runtime path does).

## Risks & mitigations

| Risk | Mitigation |
|---|---|
| awk/sed semantics drift | Differential golden tests against the Rust oracle; the spike already nailed the subtle cases (lines() newline handling, `n>=lim` exit, ultra `substr`). |
| Regex dialect mismatch | RE2 both sides; verified in spike. New filters: golden test catches any divergence. |
| Field-splitting edge cases (degenerate rows) | Per-op unit tests with short/empty rows; `strings.Fields` matches awk's whitespace splitting. |
| Future filter needs an op we didn't port | Engine ports the full op set, not a git subset; adding a filter is data + a ShellRunner. |

## Spike result (already done)

`git` filter, 4 subcommands × 3 levels = 12 golden cases, **byte-for-byte identical** to
`lowfat filter` (Rust). Negative controls confirmed the harness detects mismatches and
that the native awk/sed paths are genuinely exercised (not bypassed). The hardest filter
(the only one using `split` + the gnarliest awk state machine) passed, de-risking the rest.
