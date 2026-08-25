// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builders // import "go.opentelemetry.io/collector/service/internal/builders"

import (
	"errors"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/componentalias"
)

var (
	errNilNextConsumer = errors.New("nil next Consumer")
	NopType            = component.MustNewType("nop")
)

// logStabilityLevel logs the stability level of a component. The log level is set to info for
// undefined, unmaintained, deprecated and development. The log level is set to debug
// for alpha, beta and stable.
func logStabilityLevel(logger *zap.Logger, sl component.StabilityLevel) {
	if sl >= component.StabilityLevelAlpha {
		logger.Debug(sl.LogMessage())
	} else {
		logger.Info(sl.LogMessage())
	}
}

// logDeprecatedTypeAlias checks if the provided type is a deprecated alias and logs a warning if so.
func logDeprecatedTypeAlias(logger *zap.Logger, factory component.Factory, usedType component.Type) {
	tah, ok := factory.(componentalias.TypeAliasHolder)
	if !ok {
		return
	}
	alias := tah.DeprecatedAlias()
	if alias.String() != "" && usedType == alias {
		logger.Warn(fmt.Sprintf("%q alias is deprecated; use %q instead", alias.String(), factory.Type().String()))
	}
}
