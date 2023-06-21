// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package debugexporter // import "go.opentelemetry.io/collector/exporter/debugexporter"

// knownSyncError returns true if the given error is one of the known
// non-actionable errors returned by Sync on Plan 9.
func knownSyncError(err error) bool {
	return false
}
