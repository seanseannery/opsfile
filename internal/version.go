package internal

// Version and Commit are overwritten at build time via ldflags as defined in the Makefile:
//
//	go build -ldflags="-X sean_seannery/opsfile/internal.Version=v1.2.3 -X sean_seannery/opsfile/internal.Commit=abc1234" ./cmd/ops
var Version = "0.0.0-dev"
var Commit = "none"
