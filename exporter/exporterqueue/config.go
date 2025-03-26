// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Deprecated: [v0.123.0] Use exporterhelper.QueueConfig
type Config = exporterhelper.QueueConfig

// Deprecated: [v0.123.0] Use exporterhelper.NewDefaultQueueConfig.
// Small difference that this is blocking vs the other one which is not blocking.
func NewDefaultConfig() Config {
	return Config{
		Enabled:      true,
		Sizer:        exporterhelper.RequestSizerTypeRequests,
		NumConsumers: 10,
		QueueSize:    1_000,
		Blocking:     true,
	}
}
