// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

// TestZPagesExtensionHost verifies that the host provided by the service package
// is valid for registering the zPages extension. It starts a service with the
// zpages extension configured on a randomly-selected port and confirms that the
// service's zPages endpoint (/debug/servicez) responds successfully.
func TestZPagesExtensionHost(t *testing.T) {
	zpagesAddr := "localhost:" + getFreePort(t)

	receiverFactory := receivertest.NewNopFactory()
	exporterFactory := exportertest.NewNopFactory()
	zpagesID := component.MustNewID("zpages")

	set := service.Settings{
		BuildInfo:     component.NewDefaultBuildInfo(),
		CollectorConf: confmap.New(),
		ReceiversConfigs: map[component.ID]component.Config{
			component.NewID(nopType): receiverFactory.CreateDefaultConfig(),
		},
		ReceiversFactories: map[component.Type]receiver.Factory{
			nopType: receiverFactory,
		},
		ExportersConfigs: map[component.ID]component.Config{
			component.NewID(nopType): exporterFactory.CreateDefaultConfig(),
		},
		ExportersFactories: map[component.Type]exporter.Factory{
			nopType: exporterFactory,
		},
		ExtensionsConfigs: map[component.ID]component.Config{
			zpagesID: &zpagesextension.Config{
				ServerConfig: confighttp.ServerConfig{
					NetAddr: confignet.AddrConfig{
						Endpoint:  zpagesAddr,
						Transport: confignet.TransportTypeTCP,
					},
				},
			},
		},
		ExtensionsFactories: map[component.Type]extension.Factory{
			zpagesID.Type(): zpagesextension.NewFactory(),
		},
		TelemetryFactory: otelconftelemetry.NewFactory(),
	}

	cfg := service.Config{
		Telemetry: &otelconftelemetry.Config{
			Logs: otelconftelemetry.LogsConfig{
				Level:            zapcore.InfoLevel,
				Encoding:         "console",
				OutputPaths:      []string{"stderr"},
				ErrorOutputPaths: []string{"stderr"},
			},
			Metrics: otelconftelemetry.MetricsConfig{
				Level: configtelemetry.LevelNone,
			},
		},
		Extensions: extensions.Config{zpagesID},
		Pipelines: pipelines.Config{
			pipeline.NewID(pipeline.SignalTraces): {
				Receivers: []component.ID{component.NewID(nopType)},
				Exporters: []component.ID{component.NewID(nopType)},
			},
		},
	}

	srv, err := service.New(t.Context(), set, cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(t.Context()))
	t.Cleanup(func() {
		require.NoError(t, srv.Shutdown(t.Context()))
	})

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := http.Get("http://" + zpagesAddr + "/debug/servicez")
		if !assert.NoError(c, err) {
			return
		}
		defer resp.Body.Close()
		assert.Equal(c, http.StatusOK, resp.StatusCode)
	}, 10*time.Second, 100*time.Millisecond, "zpages servicez endpoint is not healthy")
}
