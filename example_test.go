package lowfat_test

import (
	"fmt"

	"go.rimmer.net/lowfat"
	_ "go.rimmer.net/lowfat/filters/all" // register built-in filters
)

// Example shows the typical embedding: an agent/harness that has already run a
// command and captured its output, args, and exit code, then compacts the
// output before sending it to an LLM. This is exactly the shape scout needs.
func Example() {
	// Pretend `git status` produced this (an agent would capture it from a sandbox).
	output := "On branch main\n" +
		"Changes not staged for commit:\n" +
		"\tmodified:   internal/agent/loop.go\n" +
		"\tmodified:   internal/agent/state.go\n" +
		"Untracked files:\n" +
		"\tinternal/agent/loop_test.go\n"

	res, err := lowfat.Filter("git", []string{"status"}, output,
		lowfat.Options{ExitCode: 0}.At(lowfat.Ultra))
	if err != nil {
		panic(err)
	}

	fmt.Printf("filtered=%v by=%s\n", res.Filtered, res.FilterName)
	fmt.Print(res.Output)
	// Output:
	// filtered=true by=git-compact
	// 	modified:   internal/agent/loop.go
	// 	modified:   internal/agent/state.go
	// 	internal/agent/loop_test.go
}

// ExampleRegistry shows building a registry with only a chosen subset of filters
// instead of importing filters/all.
func ExampleRegistry() {
	reg := lowfat.NewRegistry()
	// In real code: reg.MustRegister(git.New()); reg.MustRegister(docker.New())
	// Here we just show that an unknown command passes through untouched.
	res, _ := reg.Filter("python", []string{"script.py"}, "traceback...\n", lowfat.FullLevel())
	fmt.Printf("filtered=%v\n", res.Filtered)
	// Output:
	// filtered=false
}
