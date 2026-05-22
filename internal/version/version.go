package version

// Version is the current version of ocgt
// This value can be overridden at build time using -ldflags:
//   go build -ldflags "-X github.com/ethan-blue/open-code-go-tools/internal/version.Version=0.1.7"
var Version = "dev"
