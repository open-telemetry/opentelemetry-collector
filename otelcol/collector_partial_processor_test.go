// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/telemetrytest"
)

type countingProcessorConfig struct {
	Generation int `mapstructure:"generation"`
}

// countingProcessorFactories returns nop factories plus a processor that
// counts how many times it is created.
func countingProcessorFactories(t *testing.T, creates *atomic.Int32) Factories {
	t.Helper()
	factories, err := nopFactories()
	require.NoError(t, err)

	procType := component.MustNewType("countingproc")
	factories.Processors, err = MakeFactoryMap(processor.NewFactory(
		procType,
		func() component.Config { return &countingProcessorConfig{} },
		processor.WithTraces(func(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Traces) (processor.Traces, error) {
			creates.Add(1)
			return &countingProcessor{Traces: next}, nil
		}, component.StabilityLevelStable),
	))
	require.NoError(t, err)
	return factories
}

type countingProcessor struct {
	consumer.Traces
	component.StartFunc
	component.ShutdownFunc
}

func (*countingProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func processorReloadConf(processorGeneration, exporterGeneration int, extraReceiver bool) map[string]any {
	exporters := map[string]any{"nop": map[string]any{}}
	// An unused exporter changes the exporters section without touching a pipeline.
	for i := range exporterGeneration {
		exporters["nop/"+string(rune('a'+i))] = map[string]any{}
	}
	receivers := map[string]any{"nop": map[string]any{}}
	pipelineReceivers := []any{"nop"}
	if extraReceiver {
		receivers["nop/b"] = map[string]any{}
		pipelineReceivers = append(pipelineReceivers, "nop/b")
	}
	return map[string]any{
		"receivers":  receivers,
		"processors": map[string]any{"countingproc": map[string]any{"generation": processorGeneration}},
		"exporters":  exporters,
		"service": map[string]any{
			"pipelines": map[string]any{
				"traces": map[string]any{
					"receivers":  pipelineReceivers,
					"processors": []any{"countingproc"},
					"exporters":  []any{"nop"},
				},
			},
		},
	}
}

func hasLog(logs *observer.ObservedLogs, msg string) bool {
	for _, entry := range logs.All() {
		if entry.Message == msg {
			return true
		}
	}
	return false
}

func TestCollectorPartialProcessorReload(t *testing.T) {
	tests := []struct {
		name          string
		receiverGate  bool
		processorGate bool
		// Config served after the initial one.
		newProcessorGeneration int
		newExporterGeneration  int
		newExtraReceiver       bool
		wantPartial            bool
	}{
		{
			name:                   "processor_change_reloads_partially",
			processorGate:          true,
			newProcessorGeneration: 1,
			wantPartial:            true,
		},
		{
			name:                   "receiver_and_processor_change_reload_partially_together",
			receiverGate:           true,
			processorGate:          true,
			newProcessorGeneration: 1,
			newExtraReceiver:       true,
			wantPartial:            true,
		},
		{
			name:                   "receiver_and_processor_change_with_receiver_gate_disabled_reload_fully",
			processorGate:          true,
			newProcessorGeneration: 1,
			newExtraReceiver:       true,
			wantPartial:            false,
		},
		{
			name:                   "processor_and_exporter_change_reloads_fully",
			receiverGate:           true,
			processorGate:          true,
			newProcessorGeneration: 1,
			newExporterGeneration:  1,
			wantPartial:            false,
		},
		{
			name:                   "processor_gate_disabled_reloads_fully",
			receiverGate:           true,
			newProcessorGeneration: 1,
			wantPartial:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := featuregate.GlobalRegistry()
			require.NoError(t, reg.Set("service.partialReload", true))
			require.NoError(t, reg.Set("service.partialReloadReceivers", tt.receiverGate))
			require.NoError(t, reg.Set("service.partialReloadProcessors", tt.processorGate))
			defer func() {
				require.NoError(t, reg.Set("service.partialReload", false))
				require.NoError(t, reg.Set("service.partialReloadReceivers", true))
				require.NoError(t, reg.Set("service.partialReloadProcessors", true))
			}()

			observerCore, observedLogs := observer.New(zapcore.InfoLevel)
			var creates atomic.Int32
			factories := countingProcessorFactories(t, &creates)
			factories.Telemetry = telemetry.NewFactory(
				func() component.Config { return fakeTelemetryConfig{} },
				telemetrytest.WithLogger(zap.New(observerCore), nil),
			)

			var reloaded atomic.Bool
			var watcher confmap.WatcherFunc
			provider := newFakeProvider("file", func(_ context.Context, _ string, w confmap.WatcherFunc) (*confmap.Retrieved, error) {
				watcher = w
				if reloaded.Load() {
					return confmap.NewRetrieved(processorReloadConf(tt.newProcessorGeneration, tt.newExporterGeneration, tt.newExtraReceiver))
				}
				return confmap.NewRetrieved(processorReloadConf(0, 0, false))
			})

			col, err := NewCollector(CollectorSettings{
				BuildInfo: component.NewDefaultBuildInfo(),
				Factories: func() (Factories, error) { return factories, nil },
				ConfigProviderSettings: ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						URIs:              []string{"file:cfg"},
						ProviderFactories: []confmap.ProviderFactory{provider},
					},
				},
			})
			require.NoError(t, err)

			wg := startCollector(context.Background(), t, col)
			assert.Eventually(t, func() bool {
				return StateRunning == col.GetState()
			}, 2*time.Second, 50*time.Millisecond)
			require.EqualValues(t, 1, creates.Load())

			reloaded.Store(true)
			watcher(&confmap.ChangeEvent{})

			wantMsg := "Config updated, restart service"
			if tt.wantPartial {
				wantMsg = "Config updated, performing partial reload"
			}
			assert.Eventually(t, func() bool { return hasLog(observedLogs, wantMsg) }, 2*time.Second, 50*time.Millisecond)
			assert.Eventually(t, func() bool {
				return StateRunning == col.GetState() && creates.Load() == 2
			}, 2*time.Second, 50*time.Millisecond)
			assert.Equal(t, !tt.wantPartial, hasLog(observedLogs, "Config updated, restart service"))
			assert.Equal(t, tt.wantPartial, hasLog(observedLogs, "Config updated, performing partial reload"))

			col.Shutdown()
			wg.Wait()
			assert.Equal(t, StateClosed, col.GetState())
		})
	}
}
