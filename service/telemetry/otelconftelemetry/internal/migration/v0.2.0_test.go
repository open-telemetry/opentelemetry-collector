// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalLogsConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_logs.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV030{
		Encoding: "console",
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLP.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLP.Endpoint)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, "console", cfg.Encoding)
}

func TestUnmarshalTracesConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_traces.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV030{
		Level: configtelemetry.LevelNone,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLP.Endpoint)
	// check the endpoint is prefixed w/ http
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLP.Endpoint)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelNone, cfg.Level)
}

func TestUnmarshalMetricsConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{
		Level: configtelemetry.LevelBasic,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 2)
	// check the endpoint is prefixed w/ https
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Readers[0].Periodic.Exporter.OTLP.Endpoint)
	require.ElementsMatch(t, []config.NameStringValuePair{{Name: "key1", Value: ptr("value1")}, {Name: "key2", Value: ptr("value2")}}, cfg.Readers[0].Periodic.Exporter.OTLP.Headers)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelBasic, cfg.Level)
}

func ptr[T any](v T) *T {
	return &v
}
