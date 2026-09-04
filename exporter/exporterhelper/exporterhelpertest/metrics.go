// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelpertest // import "go.opentelemetry.io/collector/exporter/exporterhelper/exporterhelpertest"

import "go.opentelemetry.io/collector/exporter/exporterhelper"

// NewNop returns ObsMetrics whose operations report nothing.
func NewNop() exporterhelper.ObsMetrics {
	return exporterhelper.NewObsMetrics()
}

// NewErr returns ObsMetrics whose observer registrations return err.
func NewErr(err error) exporterhelper.ObsMetrics {
	queueMetrics := exporterhelper.NewQueueMetrics(
		exporterhelper.WithRegisterQueueSize(func(exporterhelper.Int64Value) error {
			return err
		}),
		exporterhelper.WithRegisterQueueCapacity(func(exporterhelper.Int64Value) error {
			return err
		}),
	)
	return exporterhelper.NewObsMetrics(
		exporterhelper.WithQueueBatchMetrics(exporterhelper.NewQueueBatchMetrics(
			exporterhelper.WithQueueMetrics(queueMetrics),
		)),
	)
}
