package kubectl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jrimmer/lowfat-go/lf"
)

func TestCleanYAMLPrunesServerNoise(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("..", "..", "testdata", "kubectl", "samples", "kubectl-get-yaml.txt"))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	got := cleanYAML(string(input))
	want := `apiVersion: v1
kind: Pod
metadata:
  name: web-1
  namespace: default
  annotations: <3 entries>
spec:
  containers:
  - name: web
    image: nginx:latest
status:
  phase: Running`
	if got != want {
		t.Errorf("clean-yaml mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCleanYAMLPassesThroughNonYAML(t *testing.T) {
	// A kubectl get table has none of the dropped keys -> unchanged.
	table := "NAME    READY   STATUS    RESTARTS   AGE\nweb-1   1/1     Running   0          2d\n"
	if got := cleanYAML(table); got != "NAME    READY   STATUS    RESTARTS   AGE\nweb-1   1/1     Running   0          2d" {
		t.Errorf("table should pass through unchanged, got %q", got)
	}
}

func TestShellRunnerRejectsUnknown(t *testing.T) {
	if _, err := (shell{}).RunShell("rm -rf /", "x", &lf.ExecCtx{}); err == nil {
		t.Error("unrecognized command should error")
	}
}
