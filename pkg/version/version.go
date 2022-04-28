package version

import (
	"fmt"
	"runtime"
)

var (
	versionFromGit = "unknown"
	commitFromGit  = "unknown"
	buildDate      = "unknown"
)

func MakeVersionString() string {
	return fmt.Sprintf("node-observability-agent version: %q, commit: %q, build date: %q, go version: %q, GOOS: %q, GOARCH: %q",
		versionFromGit, commitFromGit, buildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
