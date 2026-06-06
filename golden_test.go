package lowfat_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jrimmer/lowfat-go"
	_ "github.com/jrimmer/lowfat-go/filters/all"
)

// case_ is one row of testdata/cases.tsv.
type case_ struct {
	command, samplePath, args, level string
	exit                             int
	goldenPath                       string
}

func loadCases(t testing.TB) []case_ {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "cases.tsv"))
	if err != nil {
		t.Fatalf("open cases.tsv (run scripts/regen-golden.sh): %v", err)
	}
	defer f.Close()

	var cases []case_
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) != 6 {
			t.Fatalf("malformed cases.tsv row (want 6 cols): %q", line)
		}
		exit, err := strconv.Atoi(cols[4])
		if err != nil {
			t.Fatalf("bad exit code in row %q: %v", line, err)
		}
		cases = append(cases, case_{
			command:    cols[0],
			samplePath: cols[1],
			args:       cols[2],
			level:      cols[3],
			exit:       exit,
			goldenPath: cols[5],
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no cases found in cases.tsv")
	}
	return cases
}

// TestGoldenParity asserts the Go registry produces byte-for-byte identical
// output to the Rust lowfat reference for every case in cases.tsv.
func TestGoldenParity(t *testing.T) {
	cases := loadCases(t)
	reg := lowfat.Default()

	for _, c := range cases {
		c := c
		name := strings.TrimSuffix(filepath.Base(c.goldenPath), ".txt")
		t.Run(name, func(t *testing.T) {
			level, err := lowfat.ParseLevelString(c.level)
			if err != nil {
				t.Fatalf("level: %v", err)
			}
			input, err := os.ReadFile(filepath.Join("testdata", c.samplePath))
			if err != nil {
				t.Fatalf("read sample: %v", err)
			}
			want, err := os.ReadFile(filepath.Join("testdata", c.goldenPath))
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}

			// args is the full arg list; the registry derives sub = args[0].
			args := strings.Fields(c.args)
			res, err := reg.Filter(c.command, args, string(input),
				lowfat.Options{ExitCode: c.exit}.At(level))
			if err != nil {
				t.Fatalf("Filter error: %v", err)
			}
			if res.Output != string(want) {
				t.Errorf("MISMATCH command=%s args=%q level=%s exit=%d\n%s",
					c.command, c.args, c.level, c.exit, firstDiff(res.Output, string(want)))
			}
		})
	}
}

func firstDiff(got, want string) string {
	gl := strings.Split(got, "\n")
	wl := strings.Split(want, "\n")
	n := len(gl)
	if len(wl) < n {
		n = len(wl)
	}
	for i := 0; i < n; i++ {
		if gl[i] != wl[i] {
			return "first diff at line " + strconv.Itoa(i+1) +
				"\n  got:  " + quote(gl[i]) +
				"\n  want: " + quote(wl[i]) +
				"\n(got " + strconv.Itoa(len(gl)) + " lines, want " + strconv.Itoa(len(wl)) + " lines)"
		}
	}
	return "line counts differ: got " + strconv.Itoa(len(gl)) + ", want " + strconv.Itoa(len(wl))
}

func quote(s string) string {
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return strconv.Quote(s)
}
