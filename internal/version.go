package internal

// Version is the current release version of ops.
// Override at build time with:
//
//	go build -ldflags="-X sean_seannery/opsfile/internal.Version=1.2.3" ./cmd/ops/
var Version = "0.7.0"
