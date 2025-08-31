// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package otlptext // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/otlptext"

import (
	"errors"

	"golang.org/x/sys/windows"
)

// knownSyncError returns true if the given error is one of the known
// non-actionable errors returned by Sync on Windows:
//
// - sync /dev/stderr: The handle is invalid.
func knownSyncError(err error) bool {
	return errors.Is(err, windows.ERROR_INVALID_HANDLE)
}
