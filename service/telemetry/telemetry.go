// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

// NOTE TracesConfig will be removed once opentelemetry-collector-contrib
// has been updated to use otelconftelemetry instead; use at your own risk.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
type TracesConfig = migration.TracesConfigV030

// Providers is an interface for internal telemetry providers.
//
// NOTE this interface is experimental and may change in the future.
type Providers interface {
	// Shutdown gracefully shuts down the telemetry providers.
	Shutdown(context.Context) error

	// Resource returns a pcommon.Resource representing the collector.
	// This may be used by components in their internal telemetry.
	Resource() pcommon.Resource

	// Logger returns a zap.Logger that may be used by components to
	// log their internal operations.
	//
	// NOTE: from the perspective of the Telemetry implementation,
	// this Logger and the LoggerProvider are independent. However,
	// the service package will arrange for logs written to this
	// logger to be copied to the LoggerProvider. The effective
	// level of the logger will be the lower of the Logger's and
	// the LoggerProvider's levels.
	Logger() *zap.Logger

	// LoggerProvider returns a log.LoggerProvider that may be used
	// for components to log their internal operations.
	LoggerProvider() log.LoggerProvider

	// MeterProvider returns a metric.MeterProvider that may be used
	// by components to record metrics relating to their internal
	// operations.
	MeterProvider() metric.MeterProvider

	// TracerProvider returns a trace.TracerProvider that may be used
	// by components to trace their internal operations.
	TracerProvider() trace.TracerProvider
}

// Settings holds configuration for building Telemetry.
type Settings struct {
	// BuildInfo contains build information about the collector.
	BuildInfo component.BuildInfo

	// ZapOptions contains options for creating the zap logger.
	ZapOptions []zap.Option
}

// TODO create abstract Factory interface that is implemented by otelconftelemetry.
// See https://github.com/open-telemetry/opentelemetry-collector/issues/4970
