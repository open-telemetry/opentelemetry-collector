// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"errors"
	"syscall"
)

var knownSyncErrors = []error{
	// sync /dev/stdout: invalid argument
	syscall.EINVAL,
	// sync /dev/stdout: not supported
	syscall.ENOTSUP,
	// sync /dev/stdout: inappropriate ioctl for device
	syscall.ENOTTY,
	// sync /dev/stdout: bad file descriptor
	syscall.EBADF,
}

// knownSyncError returns true if the given error is one of the known
// non-actionable errors returned by Sync on Linux and macOS.
func knownSyncError(err error) bool {
	for _, syncError := range knownSyncErrors {
		if errors.Is(err, syncError) {
			return true
		}
	}

	return false
}
