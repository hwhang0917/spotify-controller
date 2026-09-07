package main

import (
	_ "embed"
	"strings"
)

// versionFile is the single source of truth for the app version; it is
// embedded at build time so the binary, both UIs and the release workflow
// (which checks the tag against it) always agree.
//
//go:embed VERSION
var versionFile string

var version = strings.TrimSpace(versionFile)
