// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// Deprecated: [v0.110.0] Use TimeoutConfig instead.
type TimeoutSettings = TimeoutConfig

type TimeoutConfig = internal.TimeoutConfig

// Deprecated: [v0.110.0] Use NewDefaultTimeoutConfig instead.
func NewDefaultTimeoutSettings() TimeoutSettings {
	return internal.NewDefaultTimeoutConfig()
}

// NewDefaultTimeoutConfig returns the default config for TimeoutConfig.
func NewDefaultTimeoutConfig() TimeoutConfig {
	return internal.NewDefaultTimeoutConfig()
}
