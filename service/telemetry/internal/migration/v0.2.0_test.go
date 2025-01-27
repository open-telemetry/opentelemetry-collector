// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalLogsConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_logs.yaml"))
	require.NoError(t, err)

	cfg := LogsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 2)
}

func TestUnmarshalTracesConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_traces.yaml"))
	require.NoError(t, err)

	cfg := TracesConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Processors, 2)
}

func TestUnmarshalMetricsConfigV020(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "v0.2.0_metrics.yaml"))
	require.NoError(t, err)

	cfg := MetricsConfigV030{}
	require.NoError(t, cm.Unmarshal(&cfg))
	require.Len(t, cfg.Readers, 2)
}
