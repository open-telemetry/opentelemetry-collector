// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/service/telemetry"
)

func newTelemetryProviders(t *testing.T, set telemetry.Settings, cfg *Config) (telemetry.Providers, *observer.ObservedLogs) {
	t.Helper()

	core, observedLogs := observer.New(zapcore.DebugLevel)
	set.ZapOptions = append(set.ZapOptions, zap.WrapCore(func(zapcore.Core) zapcore.Core { return core }))

	if len(cfg.Metrics.Readers) == 1 && cfg.Metrics.Readers[0].Pull != nil &&
		cfg.Metrics.Readers[0].Pull.Exporter.Prometheus != nil &&
		cfg.Metrics.Readers[0].Pull.Exporter.Prometheus.Port != nil &&
		*cfg.Metrics.Readers[0].Pull.Exporter.Prometheus.Port == 8888 {
		// Replace the default port with 0 to bind to an ephemeral port,
		// avoiding flaky tests.
		*cfg.Metrics.Readers[0].Pull.Exporter.Prometheus.Port = 0
	}

	factory := NewFactory()
	providers, err := factory.CreateProviders(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	t.Cleanup(func() {
		assert.NoError(t, providers.Shutdown(context.Background()))
	})
	return providers, observedLogs
}
