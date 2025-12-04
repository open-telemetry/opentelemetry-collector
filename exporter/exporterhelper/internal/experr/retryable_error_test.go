// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ----- helpers / fakes -----

// netTimeoutErr implements net.Error and returns Timeout()==true.
type netTimeoutErr struct{}

func (netTimeoutErr) Error() string   { return "i/o timeout" }
func (netTimeoutErr) Timeout() bool   { return true }
func (netTimeoutErr) Temporary() bool { return false } // kept for older interfaces

// httpRespErr is an error that carries an *http.Response and satisfies the
// minimal interface the classifier uses via errors.As.
type httpRespErr struct {
	resp *http.Response
	msg  string
}

func (e httpRespErr) Error() string            { return e.msg }
func (e httpRespErr) Response() *http.Response { return e.resp }

// lower builds a simple error with the provided message (for fallback checks).
func lower(msg string) error { return errors.New(strings.ToLower(msg)) }

// ----- tests -----

func TestClassifyErrorReason_TypedContextDeadline(t *testing.T) {
	ok, reason := ClassifyErrorReason(context.DeadlineExceeded)
	assert.True(t, ok)
	assert.Equal(t, ReasonDeadline, reason)
}

func TestClassifyErrorReason_TypedNetTimeout(t *testing.T) {
	ok, reason := ClassifyErrorReason(netTimeoutErr{})
	assert.True(t, ok)
	assert.Equal(t, ReasonNetTimeout, reason)
}

func TestClassifyErrorReason_TypedGRPC_StatusCodes(t *testing.T) {
	cases := []struct {
		err    error
		reason Reason
	}{
		{status.Error(codes.ResourceExhausted, "overloaded"), ReasonGRPCResourceExhausted},
		{status.Error(codes.Unavailable, "unavailable"), ReasonGRPCUnavailable},
		{status.Error(codes.DeadlineExceeded, "deadline"), ReasonGRPCDeadline},
	}

	for _, tc := range cases {
		ok, reason := ClassifyErrorReason(tc.err)
		assert.True(t, ok, tc.err)
		assert.Equal(t, tc.reason, reason)
	}
}

func TestClassifyErrorReason_TypedHTTP_StatusCodes(t *testing.T) {
	cases := []struct {
		code   int
		header http.Header
		reason Reason
	}{
		{http.StatusTooManyRequests, nil, ReasonHTTP429},
		{http.StatusServiceUnavailable, nil, ReasonHTTP503},
		{http.StatusGatewayTimeout, nil, ReasonHTTP504},
		{http.StatusOK, http.Header{"Retry-After": []string{"10"}}, ReasonHTTPRetryAfter},
	}

	for _, tc := range cases {
		resp := &http.Response{StatusCode: tc.code, Header: tc.header}
		err := httpRespErr{resp: resp, msg: "transport error"}
		ok, reason := ClassifyErrorReason(err)
		assert.True(t, ok, tc)
		assert.Equal(t, tc.reason, reason, tc)
	}
}

func TestClassifyErrorReason_FallbackRegex_Positive(t *testing.T) {
	// These exercise the regex fallback path.
	cases := []struct {
		err    error
		reason Reason
	}{
		{errors.New("HTTP 429 Too Many Requests"), ReasonHTTP429},
		{lower("too many requests"), ReasonHTTP429},
		{errors.New("some 429 error"), ReasonHTTP429},

		{errors.New("received HTTP 503 Service Unavailable"), ReasonHTTP503},
		{lower("service unavailable"), ReasonHTTP503},
		{errors.New("503 blah"), ReasonHTTP503},

		{errors.New("HTTP 504 Gateway Timeout"), ReasonHTTP504},
		{lower("gateway timeout"), ReasonHTTP504},
		{lower("upstream timeout"), ReasonHTTP504},
		{errors.New("got a 504"), ReasonHTTP504},

		{lower("retry-after header present"), ReasonHTTPRetryAfter},
		{errors.New("Retry-After: 30"), ReasonHTTPRetryAfter},

		{lower("circuit breaker open"), ReasonTextCircuitBreaker},
		{errors.New("Circuit Breaker"), ReasonTextCircuitBreaker},

		{lower("system overload"), ReasonTextOverload},
		{errors.New("OVERLOAD"), ReasonTextOverload},

		{lower("backpressure detected"), ReasonTextBackpressure},
		{errors.New("BACKPRESSURE"), ReasonTextBackpressure},
	}

	for _, tc := range cases {
		ok, reason := ClassifyErrorReason(tc.err)
		assert.True(t, ok, tc.err)
		assert.Equal(t, tc.reason, reason, tc.err)
	}
}

func TestIsRetryableError_Positive_Smoke(t *testing.T) {
	// A quick smoke test through the convenience wrapper.
	assert.True(t, IsRetryableError(status.Error(codes.ResourceExhausted, "x")))
	assert.True(t, IsRetryableError(netTimeoutErr{}))
	assert.True(t, IsRetryableError(context.DeadlineExceeded))
	assert.True(t, IsRetryableError(errors.New("429")))
	assert.True(t, IsRetryableError(errors.New("service unavailable")))
	assert.True(t, IsRetryableError(errors.New("gateway timeout")))
	assert.True(t, IsRetryableError(errors.New("OVERLOAD")))
}

func TestIsRetryableError_NegativeCases(t *testing.T) {
	cases := []error{
		nil,
		errors.New("some transient error"),
		errors.New("internal server error"), // intentionally not treated as retryable
		errors.New("permission denied"),     // not retryable
		errors.New("bad gateway (502)"),     // not in our fallback allowlist
		errors.New("request timeout (408)"), // not in our fallback allowlist
		errors.New("proxy timeout (599)"),   // not in our fallback allowlist
	}

	for _, err := range cases {
		assert.False(t, IsRetryableError(err), "expected not retryable: %v", err)
	}
}
