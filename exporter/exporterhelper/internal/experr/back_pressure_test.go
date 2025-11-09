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

func TestClassifyBackpressure_TypedContextDeadline(t *testing.T) {
	ok, reason := ClassifyBackpressure(context.DeadlineExceeded)
	assert.True(t, ok)
	assert.Equal(t, ReasonDeadline, reason)
}

func TestClassifyBackpressure_TypedNetTimeout(t *testing.T) {
	ok, reason := ClassifyBackpressure(netTimeoutErr{})
	assert.True(t, ok)
	assert.Equal(t, ReasonNetTimeout, reason)
}

func TestClassifyBackpressure_TypedGRPC_StatusCodes(t *testing.T) {
	cases := []struct {
		err    error
		reason Reason
	}{
		{status.Error(codes.ResourceExhausted, "overloaded"), ReasonGRPCResourceExhausted},
		{status.Error(codes.Unavailable, "unavailable"), ReasonGRPCUnavailable},
		{status.Error(codes.DeadlineExceeded, "deadline"), ReasonGRPCDeadline},
	}

	for _, tc := range cases {
		ok, reason := ClassifyBackpressure(tc.err)
		assert.True(t, ok, tc.err)
		assert.Equal(t, tc.reason, reason)
	}
}

func TestClassifyBackpressure_TypedHTTP_StatusCodes(t *testing.T) {
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
		ok, reason := ClassifyBackpressure(err)
		assert.True(t, ok, tc)
		assert.Equal(t, tc.reason, reason, tc)
	}
}

func TestClassifyBackpressure_FallbackStrings_Positive(t *testing.T) {
	// These exercise the lowercase + contains path.
	cases := []struct {
		err    error
		reason Reason
	}{
		{errors.New("HTTP 429 Too Many Requests"), ReasonHTTP429},
		{lower("too many requests"), ReasonHTTP429},

		{errors.New("received HTTP 503 Service Unavailable"), ReasonHTTP503},
		{lower("service unavailable"), ReasonHTTP503},

		{errors.New("HTTP 504 Gateway Timeout"), ReasonHTTP504},
		{lower("gateway timeout"), ReasonHTTP504},
		{lower("upstream timeout"), ReasonHTTP504},

		{lower("retry-after header present"), ReasonHTTPRetryAfter},

		{lower("circuit breaker open"), ReasonTextBackpressure},
		{lower("system overload"), ReasonTextBackpressure},
		{lower("backpressure detected"), ReasonTextBackpressure},
	}

	for _, tc := range cases {
		ok, reason := ClassifyBackpressure(tc.err)
		assert.True(t, ok, tc.err)
		assert.Equal(t, tc.reason, reason, tc.err)
	}
}

func TestIsBackpressure_Positive_Smoke(t *testing.T) {
	// A quick smoke test through the convenience wrapper.
	assert.True(t, IsBackpressure(status.Error(codes.ResourceExhausted, "x")))
	assert.True(t, IsBackpressure(netTimeoutErr{}))
	assert.True(t, IsBackpressure(context.DeadlineExceeded))
	assert.True(t, IsBackpressure(errors.New("429")))
	assert.True(t, IsBackpressure(errors.New("service unavailable")))
	assert.True(t, IsBackpressure(errors.New("gateway timeout")))
}

func TestIsBackpressure_NegativeCases(t *testing.T) {
	cases := []error{
		nil,
		errors.New("some transient error"),
		errors.New("internal server error"), // intentionally not treated as backpressure
		errors.New("permission denied"),     // not backpressure
		errors.New("bad gateway (502)"),     // not in our fallback allowlist
		errors.New("request timeout (408)"), // not in our fallback allowlist
		errors.New("proxy timeout (599)"),   // not in our fallback allowlist
	}

	for _, err := range cases {
		assert.False(t, IsBackpressure(err), "expected not backpressure: %v", err)
	}
}
