// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package runtimeinfo // import "go.opentelemetry.io/collector/service/internal/runtimeinfo"

import (
	"runtime"
	"time"
)

var (
	// InfoVar is a singleton instance of the Info struct.
	runtimeInfoVar [][2]string
)

func init() {
	runtimeInfoVar = [][2]string{
		{"StartTimestamp", time.Now().String()},
		{"Go", runtime.Version()},
		{"OS", runtime.GOOS},
		{"Arch", runtime.GOARCH},
		// Add other valuable runtime information here.
	}
}

// Info returns the runtime information like uptime, os, arch, etc.
func Info() [][2]string {
	return runtimeInfoVar
}
