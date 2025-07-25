// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer // import "go.opentelemetry.io/collector/service/internal/obsconsumer"

import (
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// Settings defines the settings for telemetry in the obsconsumer package.
type Settings struct {
	// ItemCounter is the metric to count the number of items processed.
	ItemCounter metric.Int64Counter

	// SizeCounter is the metric to count the size of items processed.
	SizeCounter metric.Int64Counter

	// Logger is the logger for the obsconsumer package.
	Logger *zap.Logger
}
