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

// TODO remove this once opentelemetry-collector-contrib is updated
// to use otelconftelemetry instead.
type TracesConfig = migration.TracesConfigV030
