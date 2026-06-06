// Package all registers every built-in lowfat filter into the default registry.
// Import it for its side effects to make all bundled tool filters available:
//
//	import _ "go.rimmer.net/lowfat/filters/all"
//
// To register only a subset, import the individual filter packages instead (each
// self-registers via init), or call their New() into a custom Registry.
package all

import (
	_ "go.rimmer.net/lowfat/filters/docker"
	_ "go.rimmer.net/lowfat/filters/find"
	_ "go.rimmer.net/lowfat/filters/git"
	_ "go.rimmer.net/lowfat/filters/grep"
	_ "go.rimmer.net/lowfat/filters/ls"
	_ "go.rimmer.net/lowfat/filters/tree"
)
