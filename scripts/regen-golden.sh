#!/usr/bin/env bash
# Regenerate golden parity fixtures from the REAL Rust lowfat binary (the oracle).
# Each filter's shipped .lf (filters/<dir>/filter.lf) is the source of truth — the
# oracle runs that exact file, and golden_test.go asserts the Go engine matches.
#
# Produces:
#   testdata/<dir>/golden/<sample>.<sub>.<level>.e<exit>.txt
#   testdata/cases.tsv   (read by golden_test.go and bench_test.go)
#
# cases.tsv columns (tab-separated):
#   command  sampleRel  args  level  exit  goldenRel
#     command   registry key (the invoked binary, e.g. "go")
#     args      full arg list incl. subcommand; sub = args[0] (may be empty)
#
# Requires a built `lowfat`. Override with LOWFAT_BIN=...
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"

LOWFAT_BIN="${LOWFAT_BIN:-../lowfat/target/release/lowfat}"
if [[ ! -x "$LOWFAT_BIN" ]]; then
  echo "error: lowfat binary not found at $LOWFAT_BIN" >&2
  echo "build it: (cd ../lowfat && cargo build --release)  or set LOWFAT_BIN=..." >&2
  exit 1
fi

CASES="testdata/cases.tsv"; : > "$CASES"

# gen <dir> <sample> <args> <exit> [command]
# command defaults to <dir>; generates ultra/lite/full goldens.
gen() {
  local dir="$1" sample="$2" args="$3" exit_code="$4" cmd="${5:-$1}"
  local lf="filters/$dir/filter.lf"
  local samplePath="testdata/$dir/samples/$sample"
  local base="${sample%.txt}"
  local sub="${args%% *}"   # first token (subcommand); "" when args is empty
  [[ "$args" == "" ]] && sub=""
  for level in ultra lite full; do
    local golden="testdata/$dir/golden/${base}.${sub}.${level}.e${exit_code}.txt"
    "$LOWFAT_BIN" filter "$lf" --sub="$sub" --level="$level" --args="$args" --exit="$exit_code" \
      < "$samplePath" > "$golden" 2>/dev/null
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
      "$cmd" "$dir/samples/$sample" "$args" "$level" "$exit_code" \
      "$dir/golden/${base}.${sub}.${level}.e${exit_code}.txt" >> "$CASES"
  done
}

# ── original six ───────────────────────────────────────────────────
gen git    git-diff-full.txt     "diff"    0
gen git    git-log-full.txt      "log"     0
gen git    git-show-full.txt     "show"    0
gen git    git-status-full.txt   "status"  0
gen git    git-status-empty.txt  "status"  0
gen docker docker-ps-full.txt      "ps"      0
gen docker docker-images-full.txt  "images"  0
gen docker docker-logs-full.txt    "logs"    0
gen docker docker-build-full.txt   "build"   0
gen docker docker-pull-full.txt    "pull"    0
gen docker docker-compose-full.txt "compose" 0
gen ls   ls-output-full.txt   "-la" 0
gen tree tree-output-full.txt "."   0
gen tree tree-error.txt       "."   2
gen grep grep-output-full.txt "-r"  0
gen grep grep-empty.txt       "-r"  1
gen grep grep-error.txt       "-r"  2
gen find find-output-full.txt "."   0
gen find find-empty.txt       "."   0

# ── tier 1 ports ───────────────────────────────────────────────────
gen gotool go-build-full.txt "build" 0 go
gen gotool go-test-full.txt  "test"  0 go
gen gotool go-mod-full.txt   "mod"   0 go
gen npm npm-install-full.txt "install" 0
gen npm npm-test-full.txt    "test"    0
gen cargo cargo-build-full.txt  "build" 0
gen cargo cargo-build-clean.txt "build" 0   # clean build -> native "cargo build: ok"
gen cargo cargo-test-full.txt   "test"  0
gen kubectl kubectl-logs-full.txt   "logs"   0
gen kubectl kubectl-events-full.txt "events" 0
gen kubectl kubectl-get-json.txt    "get pods -o json" 0   # raw path (no clean-yaml)

# ── tier 2 authored ────────────────────────────────────────────────
gen pytest pytest-fail-full.txt "tests" 1
gen pytest pytest-pass-full.txt "tests" 0
gen jest jest-full.txt "" 1
gen pip pip-install-full.txt "install requests"        0
gen pip pip-install-fail.txt "install nonexistent-pkg" 1
gen rg rg-full.txt  "-r NewServer" 0
gen rg rg-empty.txt "-r zzz"       1
gen fd fd-full.txt  ".go"          0
gen make make-full.txt  "" 2
gen make make-clean.txt "" 0
gen terraform tf-plan-full.txt  "plan"  0
gen terraform tf-apply-full.txt "apply" 0

echo "wrote $(wc -l < "$CASES" | tr -d ' ') cases to $CASES"
