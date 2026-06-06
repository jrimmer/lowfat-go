// Package all registers every built-in lowfat filter into the default registry.
// Import it for its side effects to make all bundled tool filters available:
//
//	import _ "github.com/jrimmer/lowfat-go/filters/all"
//
// To register only a subset, import the individual filter packages instead (each
// self-registers via init), or call their New() into a custom Registry.
package all

import (
	_ "github.com/jrimmer/lowfat-go/filters/docker"
	_ "github.com/jrimmer/lowfat-go/filters/find"
	_ "github.com/jrimmer/lowfat-go/filters/git"
	_ "github.com/jrimmer/lowfat-go/filters/grep"
	_ "github.com/jrimmer/lowfat-go/filters/ls"
	_ "github.com/jrimmer/lowfat-go/filters/tree"
)
