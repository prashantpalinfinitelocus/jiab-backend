// Package version exposes the build's git commit, set via -ldflags at build
// time. It exists to answer one question conclusively during a debugging
// session: is a given deployment actually running the commit I think it is,
// instead of inferring that from how a bug does or doesn't reproduce.
package version

// GitCommit is overridden at build time via:
//
//	go build -ldflags "-X github.com/Infinite-Locus-Product/coca_cola_backend/version.GitCommit=<sha>"
//
// Stays "dev" for local builds that don't pass it.
var GitCommit = "dev"
