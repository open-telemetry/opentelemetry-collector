// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/telemetrytest"
)

func TestCollectorFatalErrorsDuringLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name               string
		startGeneration    int
		shutdownGeneration int
		reload             bool
	}{
		{name: "startup", startGeneration: 1},
		{name: "reload_shutdown", shutdownGeneration: 1, reload: true},
		{name: "reload_startup", startGeneration: 2, reload: true},
		{name: "shutdown", shutdownGeneration: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			factories, err := nopFactories()
			require.NoError(t, err)
			factories.Telemetry = telemetry.NewFactory(
				func() component.Config { return fakeTelemetryConfig{} },
				telemetrytest.WithLogger(zap.New(core), nil),
			)

			var generation, starts, shutdowns int
			var reported []error
			extType := component.MustNewType("nop")
			factories.Extensions[extType] = extension.NewFactory(
				extType,
				func() component.Config { return &struct{}{} },
				func(_ context.Context, set extension.Settings, _ component.Config) (extension.Extension, error) {
					currentGeneration := generation
					var host component.Host
					report := func() {
						fatalErr := errors.New(set.ID.String())
						reported = append(reported, fatalErr)
						componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(fatalErr))
					}
					return &nopComponent{
						StartFunc: func(_ context.Context, h component.Host) error {
							starts++
							host = h
							if currentGeneration == tt.startGeneration {
								report()
							}
							return nil
						},
						ShutdownFunc: func(context.Context) error {
							shutdowns++
							if currentGeneration == tt.shutdownGeneration {
								report()
							}
							return nil
						},
					}, nil
				},
				component.StabilityLevelStable,
			)

			provider := newFakeProvider("file", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
				cfg := newConfFromFile(t, uri[5:])
				// Separate instances each report a fatal error, so the second
				// report reaches the host even though fatal status is terminal.
				cfg["extensions"] = map[string]any{"nop": nil, "nop/second": nil}
				cfg["service"].(map[string]any)["extensions"] = []any{"nop", "nop/second"}
				return confmap.NewRetrieved(cfg)
			})
			col, err := NewCollector(CollectorSettings{
				BuildInfo: component.NewDefaultBuildInfo(),
				Factories: func() (Factories, error) {
					generation++
					return factories, nil
				},
				ConfigProviderSettings: ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						URIs:              []string{filepath.Join("testdata", "otelcol-nop.yaml")},
						ProviderFactories: []confmap.ProviderFactory{provider},
					},
				},
			})
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- col.Run(ctx) }()
			if tt.startGeneration != 1 {
				require.Eventually(t, func() bool {
					return col.GetState() == StateRunning
				}, 2*time.Second, time.Millisecond)
				if tt.reload {
					col.signalsChannel <- SIGHUP
				} else {
					col.signalsChannel <- SIGTERM
				}
			}

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(10 * time.Second):
				t.Fatal("fatal error reporting blocked the collector lifecycle")
			}
			require.NoError(t, ctx.Err(), "the collector should exit without the timeout forcing shutdown")
			assert.Equal(t, StateClosed, col.GetState())
			require.Len(t, reported, 2, "both component reports must return while the control loop is busy")
			expectedGenerations := 1
			if tt.reload {
				expectedGenerations = 2
			}
			assert.Equal(t, expectedGenerations, generation)
			assert.Equal(t, 2*expectedGenerations, starts)
			assert.Equal(t, starts, shutdowns)

			fatalLogs := logs.FilterMessage("Asynchronous error received, terminating process").All()
			if tt.name == "shutdown" {
				assert.Empty(t, fatalLogs, "the control loop has already exited")
			} else {
				require.Len(t, fatalLogs, 1)
				assert.Equal(t, reported[0].Error(), fatalLogs[0].ContextMap()["error"], "the first fatal error must be retained")
			}
		})
	}
}
