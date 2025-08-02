package version

import (
	"fmt"
	"runtime"
)

var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

func GetVersion() string {
	return fmt.Sprintf("watcheth %s", Version)
}

func GetFullVersion() string {
	return fmt.Sprintf(
		"watcheth %s\nBuild Time: %s\nCommit: %s\nGo Version: %s\nOS/Arch: %s/%s",
		Version,
		BuildTime,
		CommitHash,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}