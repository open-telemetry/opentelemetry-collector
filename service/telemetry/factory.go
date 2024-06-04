// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/telemetry/internal"
)

func createDefaultConfig() component.Config {
	return &Config{
		Logs: LogsConfig{
			Level:       zapcore.InfoLevel,
			Development: false,
			Encoding:    "console",
			Sampling: &LogsSamplingConfig{
				Enabled:    true,
				Tick:       10 * time.Second,
				Initial:    10,
				Thereafter: 100,
			},
			OutputPaths:       []string{"stderr"},
			ErrorOutputPaths:  []string{"stderr"},
			DisableCaller:     false,
			DisableStacktrace: false,
			InitialFields:     map[string]any(nil),
		},
		Metrics: MetricsConfig{
			Level:   configtelemetry.LevelNormal,
			Address: ":8888",
		},
	}
}

// Factory is a telemetry factory.
type Factory = internal.Factory

// NewFactory creates a new Factory.
func NewFactory() Factory {
	return internal.NewFactory(createDefaultConfig,
		internal.WithLogger(func(_ context.Context, set Settings, cfg component.Config) (*zap.Logger, error) {
			c := *cfg.(*Config)
			return newLogger(c.Logs, set.ZapOptions)
		}),
		internal.WithTracerProvider(func(ctx context.Context, _ Settings, cfg component.Config) (trace.TracerProvider, error) {
			c := *cfg.(*Config)
			return newTracerProvider(ctx, c)
		}),
	)
}
