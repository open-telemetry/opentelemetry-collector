// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package conslogerror // import "go.opentelemetry.io/collector/consumer/conslog/conslogerror"

import (
	"go.opentelemetry.io/collector/consumer/internal/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Logs is an error that may carry associated Log data for a subset of received data
// that failed to be processed or sent.
type Logs struct {
	consumererror.Retryable[plog.Logs]
}

// NewLogs creates a Logs that can encapsulate received data that failed to be processed or sent.
func NewLogs(err error, data plog.Logs) error {
	return Logs{
		Retryable: consumererror.Retryable[plog.Logs]{
			Err:   err,
			Entry: data,
		},
	}
}
