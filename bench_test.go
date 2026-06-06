package lowfat_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"text/tabwriter"

	"github.com/jrimmer/lowfat-go"
	_ "github.com/jrimmer/lowfat-go/filters/all"
)

// benchSink prevents the compiler from eliminating the filtered result.
var benchSink lowfat.Result

// BenchmarkFilter measures filtering throughput per (filter, sub, level) over the
// real sample corpus, and reports token counts as custom metrics:
//
//	tokens_in   raw token estimate of the command output
//	tokens_out  token estimate after filtering
//	pct_saved   100 * (1 - out/in)
//
// Run: go test -bench=BenchmarkFilter -benchmem
// (Cases with empty/error samples — 0 input tokens — are skipped.)
func BenchmarkFilter(b *testing.B) {
	cases := loadCases(b)
	reg := lowfat.Default()

	for _, c := range cases {
		input, err := os.ReadFile(filepath.Join("testdata", c.samplePath))
		if err != nil {
			b.Fatalf("read sample: %v", err)
		}
		if lowfat.EstimateTokens(string(input)) == 0 {
			continue // empty/error fixtures carry no signal for savings
		}
		level, err := lowfat.ParseLevelString(c.level)
		if err != nil {
			b.Fatal(err)
		}
		args := []string{c.sub}
		opt := lowfat.Options{ExitCode: c.exit}.At(level)
		name := fmt.Sprintf("%s/%s/%s", c.filter, c.sub, c.level)

		b.Run(name, func(b *testing.B) {
			var res lowfat.Result
			b.ReportAllocs()
			b.SetBytes(int64(len(input)))
			for i := 0; i < b.N; i++ {
				res, _ = reg.Filter(c.filter, args, string(input), opt)
			}
			benchSink = res
			in, out := float64(res.InputTokens), float64(res.OutputTokens)
			b.ReportMetric(in, "tokens_in")
			b.ReportMetric(out, "tokens_out")
			if in > 0 {
				b.ReportMetric(100*(1-out/in), "pct_saved")
			}
		})
	}
}

// TestTokenSavings prints a human-readable filtered-vs-unfiltered token table
// across the corpus. It is not a pass/fail assertion of magnitude (those live in
// the golden tests); run it with -v to see the report:
//
//	go test -run TestTokenSavings -v
func TestTokenSavings(t *testing.T) {
	cases := loadCases(t)
	reg := lowfat.Default()

	type row struct {
		name    string
		in, out int
	}
	var rows []row
	// Per-filter and grand totals (content samples only).
	perFilterIn := map[string]int{}
	perFilterOut := map[string]int{}
	var totIn, totOut int

	for _, c := range cases {
		input, err := os.ReadFile(filepath.Join("testdata", c.samplePath))
		if err != nil {
			t.Fatalf("read sample: %v", err)
		}
		level, err := lowfat.ParseLevelString(c.level)
		if err != nil {
			t.Fatal(err)
		}
		res, err := reg.Filter(c.filter, []string{c.sub}, string(input),
			lowfat.Options{ExitCode: c.exit}.At(level))
		if err != nil {
			t.Fatalf("filter: %v", err)
		}
		rows = append(rows, row{
			name: fmt.Sprintf("%s %s (%s, exit %d)", c.filter, c.sub, c.level, c.exit),
			in:   res.InputTokens,
			out:  res.OutputTokens,
		})
		// Aggregate only level=full content samples to avoid triple-counting the
		// same input across ultra/lite/full, and skip zero-token fixtures.
		if c.level == "full" && res.InputTokens > 0 {
			perFilterIn[c.filter] += res.InputTokens
			perFilterOut[c.filter] += res.OutputTokens
			totIn += res.InputTokens
			totOut += res.OutputTokens
		}
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "\nCASE\tIN(tok)\tOUT(tok)\tSAVED")
	fmt.Fprintln(w, "----\t------\t-------\t-----")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", r.name, r.in, r.out, pct(r.in, r.out))
	}
	fmt.Fprintln(w, "\nPER-FILTER (full level, content samples)\t\t\t")
	filters := make([]string, 0, len(perFilterIn))
	for f := range perFilterIn {
		filters = append(filters, f)
	}
	sort.Strings(filters)
	for _, f := range filters {
		fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", f, perFilterIn[f], perFilterOut[f], pct(perFilterIn[f], perFilterOut[f]))
	}
	fmt.Fprintf(w, "TOTAL\t%d\t%d\t%s\n", totIn, totOut, pct(totIn, totOut))
	w.Flush()
	t.Log(buf.String())
}

func pct(in, out int) string {
	if in == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*(1-float64(out)/float64(in)))
}
