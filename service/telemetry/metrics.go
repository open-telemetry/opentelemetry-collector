// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

const (
	zapKeyTelemetryAddress = "address"
	zapKeyTelemetryLevel   = "metrics level"
)

// newMeterProvider creates a new MeterProvider from Config.
func newMeterProvider(set Settings, cfg Config, disableHighCardinality bool) (metric.MeterProvider, error) {
	if cfg.Metrics.Level == configtelemetry.LevelNone || len(cfg.Metrics.Readers) == 0 {
		return noop.NewMeterProvider(), nil
	}

	if set.SDK != nil {
		return set.SDK.MeterProvider(), nil
	}
	return nil, errors.New("no sdk set")
}

// LogAboutServers logs about the servers that are serving metrics.
// func (mp *meterProvider) LogAboutServers(logger *zap.Logger, cfg MetricsConfig) {
// 	for _, server := range mp.servers {
// 		logger.Info(
// 			"Serving metrics",
// 			zap.String(zapKeyTelemetryAddress, server.Addr),
// 			zap.Stringer(zapKeyTelemetryLevel, cfg.Level),
// 		)
// 	}
// }
