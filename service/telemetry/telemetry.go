// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

// Settings holds configuration for building Telemetry.
type Settings struct {
	BuildInfo  component.BuildInfo
	ZapOptions []zap.Option
}

// TODO create abstract Telemetry interface and Factory interfaces
// that are implemented by otelconftelemetry.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970

// NOTE TracesConfig will be removed once opentelemetry-collector-contrib
// has been updated to use otelconftelemetry instead; use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type TracesConfig = migration.TracesConfigV030
