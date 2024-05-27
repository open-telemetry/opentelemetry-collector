// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/internal/consumererror"

import (
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Retryable[V ptrace.Traces | pmetric.Metrics | plog.Logs] struct {
	Err   error
	Entry V
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (err Retryable[V]) Unwrap() error {
	return err.Err
}

// Error returns the provided error message, so the struct implements the `error` interface
func (err Retryable[V]) Error() string {
	return err.Err.Error()
}

// Data returns the telemetry data that failed to be processed or sent.
func (err Retryable[V]) Data() V {
	return err.Entry
}
