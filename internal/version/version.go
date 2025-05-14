package version

// These variables are meant to be set at build time using -ldflags
// Example: go build -ldflags "-X suitop/internal/version.GitCommit=$(git rev-parse HEAD) -X suitop/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	GitCommit string = "unknown"
	BuildTime string = "unknown"
	Version   string = "0.1.0-dev" // Default version
)

func Info() string {
	return "Version: " + Version + ", Commit: " + GitCommit + ", Built: " + BuildTime
}
