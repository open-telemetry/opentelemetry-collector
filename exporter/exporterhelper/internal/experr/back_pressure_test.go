// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return false }

func TestIsBackpressure_PositiveCases(t *testing.T) {
	cases := []error{
		NewBackpressureError(errors.New("wrapped")),
		timeoutNetError{},
		errors.New("HTTP 429 Too Many Requests"),
		errors.New("received HTTP 503 Service Unavailable"),
		errors.New("http 502 bad gateway"),
		errors.New("http 504 gateway timeout"),
		errors.New("http 408 request timeout"),
		errors.New("http 599 proxy timeout"),
		errors.New("too many requests"),
		errors.New("resource_exhausted"),
		errors.New("unavailable"),
		status.Error(codes.ResourceExhausted, "overloaded"),
		status.Error(codes.Unavailable, "temporarily unavailable"),
	}

	for _, err := range cases {
		assert.True(t, IsBackpressure(err), "expected backpressure: %v", err)
	}
}

func TestIsBackpressure_NegativeCases(t *testing.T) {
	cases := []error{
		nil,
		errors.New("some transient error"),
		errors.New("internal error without overload signal"),
	}

	for _, err := range cases {
		assert.False(t, IsBackpressure(err), "expected not backpressure: %v", err)
	}
}
