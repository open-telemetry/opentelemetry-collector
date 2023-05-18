// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

// LogStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func LogStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage(), zap.String(zapStabilityKey, sl.String()))
	} else {
		logger.Info(sl.LogMessage(), zap.String(zapStabilityKey, sl.String()))
	}
}
