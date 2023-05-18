// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows
// +build windows

package loggingexporter // import "go.opentelemetry.io/collector/exporter/loggingexporter"

import "golang.org/x/sys/windows"

// knownSyncError returns true if the given error is one of the known
// non-actionable errors returned by Sync on Windows:
//
// - sync /dev/stderr: The handle is invalid.
func knownSyncError(err error) bool {
	return err == windows.ERROR_INVALID_HANDLE
}
