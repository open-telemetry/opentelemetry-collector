// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cmetricerror // import "go.opentelemetry.io/collector/consumer/cmetric/cmetricerror"

import (
	"go.opentelemetry.io/collector/consumer/internal/consumererror"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Metrics is an error that may carry associated Metrics data for a subset of received data
// that failed to be processed or sent.
type Metrics struct {
	consumererror.Retryable[pmetric.Metrics]
}

// NewMetrics creates a Metrics that can encapsulate received data that failed to be processed or sent.
func NewMetrics(err error, data pmetric.Metrics) error {
	return Metrics{
		Retryable: consumererror.Retryable[pmetric.Metrics]{
			Err:   err,
			Entry: data,
		},
	}
}
