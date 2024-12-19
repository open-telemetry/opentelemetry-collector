// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/consumer/consumererror/internal"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Retryable[V ptrace.Traces | pmetric.Metrics | plog.Logs | pprofile.Profiles] struct {
	Err   error
	Value V
}

// Error provides the error message
func (err Retryable[V]) Error() string {
	return err.Err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (err Retryable[V]) Unwrap() error {
	return err.Err
}

// Data returns the telemetry data that failed to be processed or sent.
func (err Retryable[V]) Data() V {
	return err.Value
}
