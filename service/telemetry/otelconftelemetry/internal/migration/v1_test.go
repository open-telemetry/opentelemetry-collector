// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalLogsConfigV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1_logs.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV1{
		Encoding: "console",
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	require.NotNil(t, cfg.Processors[0].Batch)
	require.NotNil(t, cfg.Processors[0].Batch.Exporter.OTLPHttp)
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLPHttp.Endpoint)
	require.NotNil(t, cfg.Processors[2].Simple)
	require.NotNil(t, cfg.Processors[2].Simple.Exporter.OTLPHttp)
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLPHttp.Endpoint)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, "console", cfg.Encoding)
}

func TestUnmarshalTracesConfigV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1_traces.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV1{
		Level: configtelemetry.LevelNone,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 3)
	require.NotNil(t, cfg.Processors[0].Batch)
	require.NotNil(t, cfg.Processors[0].Batch.Exporter.OTLPHttp)
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Processors[0].Batch.Exporter.OTLPHttp.Endpoint)
	require.NotNil(t, cfg.Processors[2].Simple)
	require.NotNil(t, cfg.Processors[2].Simple.Exporter.OTLPHttp)
	require.Equal(t, "http://127.0.0.1:4317", *cfg.Processors[2].Simple.Exporter.OTLPHttp.Endpoint)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelNone, cfg.Level)
}

func TestUnmarshalMetricsConfigV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v1_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV1{
		Level: configtelemetry.LevelBasic,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 2)
	require.NotNil(t, cfg.Readers[0].Periodic)
	require.NotNil(t, cfg.Readers[0].Periodic.Exporter.OTLPGrpc)
	require.Equal(t, "https://127.0.0.1:4317", *cfg.Readers[0].Periodic.Exporter.OTLPGrpc.Endpoint)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelBasic, cfg.Level)
}

func TestUnmarshalLogsConfigV030ToV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_logs_migration.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV1{
		Encoding: "console",
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 1)
	require.NotNil(t, cfg.Processors[0].Simple)
	require.NotNil(t, cfg.Processors[0].Simple.Exporter.Console)
	require.True(t, cfg.DisableZapResource)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, "console", cfg.Encoding)
}

func TestUnmarshalTracesConfigV030ToV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_traces_migration.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV1{
		Level: configtelemetry.LevelNone,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 1)
	require.NotNil(t, cfg.Processors[0].Simple)
	require.NotNil(t, cfg.Processors[0].Simple.Exporter.Console)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelNone, cfg.Level)
}

func TestUnmarshalMetricsConfigV030ToV1(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.3.0_metrics_migration.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV1{
		Level: configtelemetry.LevelBasic,
	}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 1)
	require.NotNil(t, cfg.Readers[0].Pull)
	require.NotNil(t, cfg.Readers[0].Pull.Exporter.PrometheusDevelopment)
	// ensure defaults set in the original config object are not lost
	require.Equal(t, configtelemetry.LevelBasic, cfg.Level)
}
