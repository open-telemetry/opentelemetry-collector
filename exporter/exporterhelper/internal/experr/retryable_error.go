// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"

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
	ReasonTextCircuitBreaker    Reason = "text:circuit_breaker"
	ReasonTextOverload          Reason = "text:overload"
	ReasonTextBackpressure      Reason = "text:backpressure"
)

var fallbackRegex = regexp.MustCompile(
	`(?i)(429|too many requests)|` + // ReasonHTTP429
		`(503|service unavailable)|` + // ReasonHTTP503
		`(504|gateway timeout|upstream timeout)|` + // ReasonHTTP504
		`(retry-after)|` + // ReasonHTTPRetryAfter
		`(circuit breaker)|` + // ReasonTextCircuitBreaker
		`(overload)|` + // ReasonTextOverload
		`(backpressure)`, // ReasonTextBackpressure
)

// ClassifyErrorReason returns (isRetryable, reason).
// The logic is strictly:
//  1. Typed signals first (fast, no allocs)
//  2. Fallback to a single regex match on the error string
//
// NOTE: This function runs only on error paths. Even then,
// typed checks return early. Fallback string scanning is a
// micro-cost compared to network I/O and retry backoff.
func ClassifyErrorReason(err error) (bool, Reason) {
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

	// 5) Fallback: cheap regex match on error string.
	matches := fallbackRegex.FindStringSubmatch(err.Error())
	if matches == nil {
		return false, ""
	}

	// The regex has 7 capture groups, one for each reason.
	// FindStringSubmatch returns the full match at [0], then groups at [1]...[n].
	// We check which group captured.
	switch {
	case matches[1] != "":
		return true, ReasonHTTP429
	case matches[2] != "":
		return true, ReasonHTTP503
	case matches[3] != "":
		return true, ReasonHTTP504
	case matches[4] != "":
		return true, ReasonHTTPRetryAfter
	case matches[5] != "":
		return true, ReasonTextCircuitBreaker
	case matches[6] != "":
		return true, ReasonTextOverload
	case matches[7] != "":
		return true, ReasonTextBackpressure
	}

	return false, ""
}

// IsRetryableError is a convenience wrapper.
func IsRetryableError(err error) bool {
	ok, _ := ClassifyErrorReason(err)
	return ok
}
