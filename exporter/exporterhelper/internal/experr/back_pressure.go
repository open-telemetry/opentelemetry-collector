// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"

import (
	"context"
	"errors"
	"net"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type backpressureError struct{ error }

func NewBackpressureError(err error) error {
	if err == nil {
		return nil
	}
	return backpressureError{err}
}

// IsBackpressure returns true if the error is an explicit backpressure error
// OR it matches well-known HTTP/gRPC backpressure signals.
func IsBackpressure(err error) bool {
	if err == nil {
		return false
	}
	// Our explicit wrapper.
	var x backpressureError
	if errors.As(err, &x) {
		return true
	}

	// Check for net.Error timeout
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}

	// Context timed out (treat as transient/backpressure for ARC)
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// gRPC: RESOURCE_EXHAUSTED (8), UNAVAILABLE (14), DEADLINE_EXCEEDED (4)
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.ResourceExhausted, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	// HTTP-ish strings (safe fallback; exporters often wrap as text)
	// NOTE: intentionally case-insensitive and substring-based.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "bad gateway") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "gateway timeout") ||
		strings.Contains(msg, "408") ||
		strings.Contains(msg, "request timeout") ||
		strings.Contains(msg, "599") ||
		strings.Contains(msg, "proxy timeout") ||
		strings.Contains(msg, "resource_exhausted") ||
		strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "deadline_exceeded") || // Added for string-based gRPC errors
		strings.Contains(msg, "internal server error") ||
		strings.Contains(msg, "circuit breaker") {
		return true
	}

	return false
}
