#!/usr/bin/env bash
# Regenerate golden parity fixtures from the REAL Rust lowfat binary (the oracle).
#
# Produces:
#   testdata/<filter>/golden/<sample>.<sub>.<level>.e<exit>.txt   (one per case)
#   testdata/cases.tsv                                            (case index, read by golden_test.go)
#
# Requires: a built `lowfat` binary. Override its path with LOWFAT_BIN=...
# Default looks for the sibling Rust checkout's release build.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

LOWFAT_BIN="${LOWFAT_BIN:-../lowfat/target/release/lowfat}"
if [[ ! -x "$LOWFAT_BIN" ]]; then
  echo "error: lowfat binary not found at $LOWFAT_BIN" >&2
  echo "build it with: (cd ../lowfat && cargo build --release)  or set LOWFAT_BIN=..." >&2
  exit 1
fi

EMB="../lowfat/crates/lowfat-plugin/embedded"
CASES="testdata/cases.tsv"
: > "$CASES"

# emit <filter> <sampleFileBasename> <sub> <exit>
# generates ultra/lite/full goldens and appends rows to cases.tsv
gen() {
  local filter="$1" sample="$2" sub="$3" exit_code="$4"
  local lf="$EMB/$filter/$filter-compact/filter.lf"
  local samplePath="testdata/$filter/samples/$sample"
  local base="${sample%.txt}"
  for level in ultra lite full; do
    local golden="testdata/$filter/golden/${base}.${sub}.${level}.e${exit_code}.txt"
    "$LOWFAT_BIN" filter "$lf" --sub="$sub" --level="$level" --exit="$exit_code" \
      < "$samplePath" > "$golden" 2>/dev/null
    # cols: filter  sampleRel  sub  level  exit  goldenRel  (paths relative to testdata/)
    printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
      "$filter" "$filter/samples/$sample" "$sub" "$level" "$exit_code" \
      "$filter/golden/${base}.${sub}.${level}.e${exit_code}.txt" >> "$CASES"
  done
}

# ── git ────────────────────────────────────────────────────────────
gen git git-diff-full.txt    diff   0
gen git git-log-full.txt     log    0
gen git git-show-full.txt    show   0
gen git git-status-full.txt  status 0
gen git git-status-empty.txt status 0     # empty -> `or "git status: clean"`

# ── docker ─────────────────────────────────────────────────────────
gen docker docker-ps-full.txt      ps      0
gen docker docker-images-full.txt  images  0
gen docker docker-logs-full.txt    logs    0
gen docker docker-build-full.txt   build   0
gen docker docker-pull-full.txt    pull    0
gen docker docker-compose-full.txt compose 0

# ── ls (pure '*') ──────────────────────────────────────────────────
gen ls ls-output-full.txt -la 0

# ── tree (pure '*'; exit-failed -> raw) ────────────────────────────
gen tree tree-output-full.txt . 0
gen tree tree-error.txt       . 2

# ── grep (exit 1 = no match, 2 = error) ────────────────────────────
gen grep grep-output-full.txt -r 0
gen grep grep-empty.txt       -r 1        # no matches -> `or "grep: no matches"`
gen grep grep-error.txt       -r 2        # error -> raw

# ── find (exit-failed -> raw) ──────────────────────────────────────
gen find find-output-full.txt . 0
gen find find-empty.txt       . 0         # empty -> `or "find: no matches"`

echo "wrote $(wc -l < "$CASES" | tr -d ' ') cases to $CASES"
