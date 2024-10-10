// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/config/configtimeout"
)

type TimeoutConfig = configtimeout.TimeoutConfig

// NewDefaultConfig returns the default config for TimeoutConfig.
func NewDefaultTimeoutConfig() TimeoutConfig {
	return configtimeout.NewDefaultConfig()
}
