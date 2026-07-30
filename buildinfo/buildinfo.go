// Package buildinfo exposes build metadata injected through linker flags.
package buildinfo

import "fmt"

var (
	// Version is the release version.
	Version = "development"
	// Commit is the source revision.
	Commit = "unknown"
	// BuildTime is the RFC 3339 build timestamp.
	BuildTime = "unknown"
)

// String returns stable, human-readable build metadata.
func String() string {
	return fmt.Sprintf("tidekeepers-api version=%s commit=%s built=%s", Version, Commit, BuildTime)
}
