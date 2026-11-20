// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package reclaim // import "go.opentelemetry.io/collector/service/internal/reclaim"

// ColdBinaryPages is a no-op on non-Linux operating systems.
func ColdBinaryPages() (Result, error) {
	return Result{SkippedReason: "unsupported operating system"}, nil
}
