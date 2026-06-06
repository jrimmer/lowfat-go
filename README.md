# lowfat (Go)

A pure-Go port of [lowfat](https://github.com/zdk/lowfat)'s command-output
filtering engine, packaged as an **embeddable library**. It compacts noisy CLI
output (git, docker, ls, tree, grep, find) before that output reaches an LLM,
cutting token cost — with **no subprocess execution** and **no dynamic linking**.

It is the engine, not the CLI: there is no `lowfat` binary, no SQLite metrics,
no `.lowfat` config files. You call a function with `(command, args, output)`
and get back compacted text plus token counts.

```go
import (
    "github.com/jrimmer/lowfat-go"
    _ "github.com/jrimmer/lowfat-go/filters/all" // register the built-in tool filters
)

res, err := lowfat.Filter("git", []string{"diff"}, output,
    lowfat.Options{ExitCode: 0}.At(lowfat.Ultra))
// res.Output  -> compacted text
// res.InputTokens - res.OutputTokens -> tokens saved
// res.Filtered, res.FilterName
```

`args` is everything after the command name; `args[0]` is the subcommand
(matching lowfat's dispatch). If no filter is registered for `command`, the
output is returned unchanged with `Filtered == false`.

## How filtering works

Each tool filter is an embedded `.lf` ruleset (lowfat's line-oriented filter DSL:
`keep`/`drop`/`head`/`tail`/`match level`/`if exit failed`/`split`/macros) plus a
pure-Go `ShellRunner` that reimplements that tool's `awk`/`sed` transforms
natively. Output is byte-for-byte identical to the Rust `lowfat filter` (see
[testing](#testing)).

| Filter | command(s) | notes |
|---|---|---|
| `filters/git` | `git` | status / diff / log / show; native `compact-diff`, `abbrev-hash` |
| `filters/docker` | `docker` | ps / images / logs / build / pull / compose; native column extraction |
| `filters/ls` | `ls` | native long-form collapse, name-only mode |
| `filters/tree` | `tree` | pure-DSL |
| `filters/grep` | `grep` | pure-DSL |
| `filters/find` | `find` | pure-DSL |
| `filters/gotool` | `go` | build / test / vet / mod (pure-DSL; package `gotool` — `go` is reserved) |
| `filters/npm` | `npm` | install / test / audit / run (pure-DSL) |
| `filters/cargo` | `cargo` | build / test / check / clippy / run / add / update; native `ok` status line |
| `filters/kubectl` | `kubectl` | get / logs / events; native indent-based YAML pruner (replaces upstream PyYAML) |
| `filters/pytest` | `pytest` | keep failures + summary, drop progress dots (pure-DSL) |
| `filters/jest` | `jest`, `vitest` | keep PASS/FAIL suites + totals (pure-DSL) |
| `filters/pip` | `pip`, `pip3` | drop "Requirement already satisfied" spam (pure-DSL) |
| `filters/rg` | `rg` | ripgrep; mirrors grep (pure-DSL) |
| `filters/fd` | `fd` | modern find; mirrors find (pure-DSL) |
| `filters/make` | `make` | keep warnings/errors, drop dir chatter (pure-DSL) |
| `filters/terraform` | `terraform`, `tofu` | plan / apply summaries (pure-DSL) |

## Adding a new tool filter (compile-time, no dynamic linking)

New tool types are added in a pluggable fashion, but they require a new compile —
there is no runtime loading. To add, say, `kubectl`:

1. `mkdir filters/kubectl`, add a `filter.lf` and:

   ```go
   package kubectl

   import (
       _ "embed"
       "github.com/jrimmer/lowfat-go"
   )

   //go:embed filter.lf
   var filterLF string

   func New() *lowfat.ToolFilter {
       return lowfat.MustNewFilter(lowfat.Manifest{
           Name:     "kubectl-compact",
           Commands: []string{"kubectl"},
       }, filterLF, shell{}) // shell{} implements lf.ShellRunner; or nil for pure-DSL
   }

   func init() { lowfat.MustRegister(New()) }
   ```

2. If the `.lf` uses `shell:` ops, implement `lf.ShellRunner` in a `shell.go`
   (reimplement the awk/sed in Go — see `filters/git/shell.go` for the pattern,
   `internal/awk` for helpers).
3. Add `_ "github.com/jrimmer/lowfat-go/filters/kubectl"` to `filters/all/all.go`.
4. Recompile.

To expose only a subset of filters, skip `filters/all` and register the
`New()` constructors you want into your own `lowfat.NewRegistry()`.

## Testing

```sh
go test ./...
```

The flagship suite is **golden parity**: `testdata/cases.tsv` enumerates 126 cases
(every filter × {ultra,lite,full}, plus `exit!=0` and empty-output paths); each
golden was produced by the **real Rust `lowfat`** binary running the shipped `.lf`,
and `golden_test.go` asserts the Go output matches byte-for-byte. (The one path the
oracle can't cover — kubectl `get -o yaml`, whose upstream pruner is PyYAML — is
unit-tested directly in `filters/kubectl/shell_test.go`.) Regenerate after changing
a filter:

```sh
(cd ../lowfat && cargo build --release)   # build the oracle
./scripts/regen-golden.sh                  # or LOWFAT_BIN=/path/to/lowfat ./scripts/regen-golden.sh
```

Plus engine unit tests (`lf/`), native-op tests (`filters/*/shell_test.go`), and
integration-API tests (`api_test.go`, `example_test.go`).

### Token savings

`bench_test.go` measures filtered-vs-unfiltered token counts over the corpus:

```sh
go test -run TestTokenSavings -v          # human-readable savings table
go test -bench BenchmarkFilter -benchmem  # ns/op + tokens_in/tokens_out/pct_saved per case
```

The benchmark reports `tokens_in`, `tokens_out`, and `pct_saved` as custom metrics
alongside throughput.

Representative results on the bundled samples (`TestTokenSavings`), tokens estimated
as `ceil(len/4)`:

| case | in | out (ultra) | saved | out (full) | saved |
|---|--:|--:|--:|--:|--:|
| git diff | 2782 | 121 | **95.7%** | 1716 | 38.3% |
| git log | 911 | 86 | **90.6%** | 178 | 80.5% |
| git show | 1193 | 116 | **90.3%** | 630 | 47.2% |
| docker images | 1468 | 207 | **85.9%** | 616 | 58.0% |
| docker ps | 271 | 41 | **84.9%** | 168 | 38.0% |
| ls -la | 169 | 23 | **86.4%** | 42 | 75.1% |
| find | 429 | 189 | **55.9%** | 429 | 0.0% |
| tree | 322 | 214 | **33.5%** | 322 | 0.0% |

`ultra` does the heavy lifting (≈85–95% on large outputs); `full` is conservative and
leaves outputs smaller than its `head` window (tree/grep/find/docker-logs samples)
untouched — filtering only pays off once output is large.

Throughput is ~100–1400 MB/s depending on filter (e.g. `git diff/ultra` ≈ 7.8 µs/op,
`docker logs` ≈ 0.35 µs/op on an M-series core). Numbers vary by machine; regenerate
with `go test -bench BenchmarkFilter -benchmem`.

## Layout

See [PORT_PLAN.md](PORT_PLAN.md) for the full design, decisions, and scope.

```
lowfat.go / manifest.go / tokens.go   integration API (Registry, Filter, Options, Result)
lf/                                   the .lf DSL engine (parser + executor)
internal/awk/                         pure-Go awk/sed primitives for ShellRunners
filters/<tool>/                       one package per tool type (pluggable)
testdata/  scripts/regen-golden.sh    golden parity harness
```

## Install

```sh
go get github.com/jrimmer/lowfat-go
```

The import path is `github.com/jrimmer/lowfat-go` (package `lowfat`); built-in
filters live under `github.com/jrimmer/lowfat-go/filters/...`.

## Credits & upstream

This is a Go port of **lowfat** by **zdk**:

- Original project & repository: <https://github.com/zdk/lowfat>

The `.lf` DSL, the bundled filter rulesets (`filters/*/filter.lf`), the test
samples under `testdata/`, and the overall design are derived from that project.
Output is verified byte-for-byte against the upstream Rust `lowfat filter`
(see [Testing](#testing)). All credit for the filtering approach and rules goes
to the original authors; this repository only re-implements the engine in Go.

The upstream project is licensed under **Apache-2.0**; this port carries the same
license. See the original repository for the canonical CLI, plugin ecosystem,
shell integration, and documentation.
