// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux
// +build !linux

package iruntime // import "go.opentelemetry.io/collector/internal/iruntime"

// TotalMemory returns total available memory for non-linux platforms.
func TotalMemory() (uint64, error) {
	return readMemInfo()
}
