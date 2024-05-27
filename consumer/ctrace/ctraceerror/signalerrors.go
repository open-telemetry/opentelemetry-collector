// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctraceerror // import "go.opentelemetry.io/collector/consumer/ctrace/ctraceerror"

import (
	"go.opentelemetry.io/collector/consumer/internal/consumererror"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Traces is an error that may carry associated Trace data for a subset of received data
// that failed to be processed or sent.
type Traces struct {
	consumererror.Retryable[ptrace.Traces]
}

// NewTraces creates a Traces that can encapsulate received data that failed to be processed or sent.
func NewTraces(err error, data ptrace.Traces) error {
	return Traces{
		Retryable: consumererror.Retryable[ptrace.Traces]{
			Err:   err,
			Entry: data,
		},
	}
}
