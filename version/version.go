// Package version exposes the build-time-injected service version.
//
// The default value of "unknown" is overridden at link time via:
//
//	go build -ldflags "-X github.com/andrewhowdencom/x40.link/version.Version=<value>"
//
// The Containerfile and Taskfile.yml `bin` task both inject this. When the
// link-time injection is absent (e.g. running tests), the value remains
// "unknown", which is acceptable per the OpenTelemetry resource attribute
// semantic conventions for service.version.
package version

// Version is the build-time-injected service version.
var Version = "unknown"
