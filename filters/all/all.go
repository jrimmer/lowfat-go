// Package all registers every built-in lowfat filter into the default registry.
// Import it for its side effects to make all bundled tool filters available:
//
//	import _ "github.com/jrimmer/lowfat-go/filters/all"
//
// To register only a subset, import the individual filter packages instead (each
// self-registers via init), or call their New() into a custom Registry.
package all

import (
	_ "github.com/jrimmer/lowfat-go/filters/cargo"
	_ "github.com/jrimmer/lowfat-go/filters/docker"
	_ "github.com/jrimmer/lowfat-go/filters/fd"
	_ "github.com/jrimmer/lowfat-go/filters/find"
	_ "github.com/jrimmer/lowfat-go/filters/git"
	_ "github.com/jrimmer/lowfat-go/filters/gotool"
	_ "github.com/jrimmer/lowfat-go/filters/grep"
	_ "github.com/jrimmer/lowfat-go/filters/jest"
	_ "github.com/jrimmer/lowfat-go/filters/kubectl"
	_ "github.com/jrimmer/lowfat-go/filters/ls"
	_ "github.com/jrimmer/lowfat-go/filters/make"
	_ "github.com/jrimmer/lowfat-go/filters/npm"
	_ "github.com/jrimmer/lowfat-go/filters/pip"
	_ "github.com/jrimmer/lowfat-go/filters/pytest"
	_ "github.com/jrimmer/lowfat-go/filters/rg"
	_ "github.com/jrimmer/lowfat-go/filters/terraform"
	_ "github.com/jrimmer/lowfat-go/filters/tree"
)
