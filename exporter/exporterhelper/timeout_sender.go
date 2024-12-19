// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

type TimeoutConfig = internal.TimeoutConfig

// NewDefaultTimeoutConfig returns the default config for TimeoutConfig.
func NewDefaultTimeoutConfig() TimeoutConfig {
	return internal.NewDefaultTimeoutConfig()
}
