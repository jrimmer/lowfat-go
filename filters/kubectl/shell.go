package kubectl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jrimmer/lowfat-go/internal/awk"
	"github.com/jrimmer/lowfat-go/lf"
)

// shell implements lf.ShellRunner for kubectl's `clean-yaml` op.
type shell struct{}

func (shell) RunShell(cmd, stdin string, ctx *lf.ExecCtx) (string, error) {
	if strings.TrimSpace(cmd) == "kube-clean-yaml" {
		return cleanYAML(stdin), nil
	}
	return "", fmt.Errorf("kubectl filter: unrecognized shell command: %q", cmd)
}

// dropKeys are server-side / bookkeeping fields whose entire (possibly nested)
// blocks are removed from `kubectl get -o yaml` output.
var dropKeys = map[string]bool{
	"managedFields":     true,
	"resourceVersion":   true,
	"generation":        true,
	"creationTimestamp": true,
	"uid":               true,
	"selfLink":          true,
	"ownerReferences":   true,
}

// cleanYAML is a native, indent-based approximation of upstream's PyYAML pruner:
// it removes the dropKeys blocks and collapses an `annotations:` map to a count.
// It is line-based (not a full YAML parse), which is sufficient for the
// server-managed noise that dominates `kubectl get -o yaml`. Non-YAML input
// (e.g. a table) has none of these keys and passes through unchanged.
func cleanYAML(input string) string {
	lines := awk.Lines(input)
	var out []string
	for i := 0; i < len(lines); {
		line := lines[i]
		indent := leadingSpaces(line)
		key := yamlKey(line)

		if dropKeys[key] {
			// Drop the key line and its block: deeper-indented children, blank
			// lines, and same-indent list items (YAML lists sit at the key's
			// indent, e.g. `managedFields:` followed by `- apiVersion: ...`).
			i++
			for i < len(lines) {
				l := lines[i]
				if isBlank(l) {
					i++
					continue
				}
				ind := leadingSpaces(l)
				if ind > indent {
					i++
					continue
				}
				if ind == indent && strings.HasPrefix(strings.TrimLeft(l, " "), "- ") {
					i++
					continue
				}
				break
			}
			continue
		}

		if key == "annotations" {
			// Collapse the block to `annotations: <N entries>`, N = direct children.
			n, next := countDirectChildren(lines, i, indent)
			out = append(out, strings.Repeat(" ", indent)+"annotations: <"+strconv.Itoa(n)+" entries>")
			i = next
			continue
		}

		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
}

// countDirectChildren counts immediate child lines of the block whose header is
// at lines[start] (indent parentIndent), and returns the index just past the
// block. Direct children are the lines at the first child indent level.
func countDirectChildren(lines []string, start, parentIndent int) (n, next int) {
	childIndent := -1
	j := start + 1
	for j < len(lines) {
		if isBlank(lines[j]) {
			j++
			continue
		}
		ind := leadingSpaces(lines[j])
		if ind <= parentIndent {
			break
		}
		if childIndent == -1 {
			childIndent = ind
		}
		if ind == childIndent {
			n++
		}
		j++
	}
	return n, j
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

// yamlKey returns the mapping key of a line ("foo" for "  foo: bar"), stripping
// an optional "- " list-item prefix. Returns "" if the line is not a key.
func yamlKey(line string) string {
	s := strings.TrimLeft(line, " ")
	s = strings.TrimPrefix(s, "- ")
	colon := strings.IndexByte(s, ':')
	if colon <= 0 {
		return ""
	}
	return s[:colon]
}
