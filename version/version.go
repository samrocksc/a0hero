package version

import (
	"fmt"
	"runtime"
)

// Set via -ldflags at build time.
var (
	Version   = "dev"    // e.g. v0.1.0
	Commit    = "none"   // git short hash
	BuildDate = "unknown"
)

// Info returns a full version string.
func Info() string {
	return fmt.Sprintf("a0hero %s (commit: %s, built: %s, %s/%s)",
		Version, Commit, BuildDate, runtime.GOOS, runtime.GOARCH)
}

// Short returns just the version tag.
func Short() string {
	return Version
}

// OSArch returns the platform string (e.g. "darwin/arm64").
func OSArch() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}