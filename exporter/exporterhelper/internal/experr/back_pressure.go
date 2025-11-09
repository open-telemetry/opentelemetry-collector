// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Reason is emitted to logs/metrics when desired.
// You don't have to use it in telemetry if your metadata builder
// doesn't have a "reason" attribute yet.
type Reason string

const (
	ReasonGRPCResourceExhausted Reason = "grpc_resource_exhausted"
	ReasonGRPCUnavailable       Reason = "grpc_unavailable"
	ReasonGRPCDeadline          Reason = "grpc_deadline_exceeded"
	ReasonDeadline              Reason = "deadline_exceeded"
	ReasonHTTP429               Reason = "http_429"
	ReasonHTTP503               Reason = "http_503"
	ReasonHTTP504               Reason = "http_504"
	ReasonHTTPRetryAfter        Reason = "http_retry_after"
	ReasonNetTimeout            Reason = "net_timeout"
	ReasonTextBackpressure      Reason = "text_backpressure"
)

// ClassifyBackpressure returns (isBackpressure, reason).
// The logic is strictly:
//  1. Typed signals first (fast, no allocs)
//  2. Fallback to a single lowercase + a few contains checks
//
// NOTE: This function runs only on error paths. Even then,
// typed checks return early. Fallback string scanning is a
// micro-cost compared to network I/O and retry backoff.
func ClassifyBackpressure(err error) (bool, Reason) {
	if err == nil {
		return false, ""
	}

	// 1) Context deadline/timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return true, ReasonDeadline
	}

	// 2) net.Error timeouts
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true, ReasonNetTimeout
	}

	// 3) gRPC statuses (common in OTLP/gRPC)
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.ResourceExhausted:
			return true, ReasonGRPCResourceExhausted
		case codes.Unavailable:
			return true, ReasonGRPCUnavailable
		case codes.DeadlineExceeded:
			return true, ReasonGRPCDeadline
		}
	}

	// 4) Wrapped HTTP response providers (if your error wraps response)
	// Define a minimal interface to avoid importing your transport layer.
	type respCarrier interface{ Response() *http.Response }
	var rc respCarrier
	if errors.As(err, &rc) {
		if r := rc.Response(); r != nil {
			switch r.StatusCode {
			case http.StatusTooManyRequests:
				return true, ReasonHTTP429
			case http.StatusServiceUnavailable:
				return true, ReasonHTTP503
			case http.StatusGatewayTimeout:
				return true, ReasonHTTP504
			}
			if r.Header.Get("Retry-After") != "" {
				return true, ReasonHTTPRetryAfter
			}
		}
	}

	// 5) Fallback: cheap lowercase + tight substring checks.
	// Keep this list intentionally small to minimize false positives.
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "429") || strings.Contains(s, "too many requests") {
		return true, ReasonHTTP429
	}
	if strings.Contains(s, "503") || strings.Contains(s, "service unavailable") {
		return true, ReasonHTTP503
	}
	if strings.Contains(s, "504") || strings.Contains(s, "gateway timeout") || strings.Contains(s, "upstream timeout") {
		return true, ReasonHTTP504
	}
	if strings.Contains(s, "retry-after") {
		return true, ReasonHTTPRetryAfter
	}
	if strings.Contains(s, "circuit breaker") || strings.Contains(s, "overload") || strings.Contains(s, "backpressure") {
		return true, ReasonTextBackpressure
	}

	return false, ""
}

// IsBackpressure is a convenience wrapper.
func IsBackpressure(err error) bool {
	ok, _ := ClassifyBackpressure(err)
	return ok
}
