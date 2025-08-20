// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"errors"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

// newMeterProvider creates a new MeterProvider from Config.
func newMeterProvider(cfg Config, sdk *config.SDK) (metric.MeterProvider, error) {
	if cfg.Metrics.Level == configtelemetry.LevelNone || len(cfg.Metrics.Readers) == 0 {
		return noop.NewMeterProvider(), nil
	}

	if sdk != nil {
		return sdk.MeterProvider(), nil
	}
	return nil, errors.New("no sdk set")
}
