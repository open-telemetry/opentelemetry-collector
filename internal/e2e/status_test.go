// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

var nopType = component.MustNewType("nop")

var wg = sync.WaitGroup{}

func Test_ComponentStatusReporting_SharedInstance(t *testing.T) {
	eventsReceived := make(map[*componentstatus.InstanceID][]*componentstatus.Event)
	exporterFactory := exportertest.NewNopFactory()
	connectorFactory := connectortest.NewNopFactory()
	// Use a different ID than receivertest and exportertest to avoid ambiguous
	// configuration scenarios. Ambiguous IDs are detected in the 'otelcol' package,
	// but lower level packages such as 'service' assume that IDs are disambiguated.
	connID := component.NewIDWithName(nopType, "conn")

	set := service.Settings{
		BuildInfo:     component.NewDefaultBuildInfo(),
		CollectorConf: confmap.New(),
		ReceiversConfigs: map[component.ID]component.Config{
			component.NewID(component.MustNewType("test")): &receiverConfig{},
		},
		ReceiversFactories: map[component.Type]receiver.Factory{
			component.MustNewType("test"): newReceiverFactory(),
		},
		ExportersConfigs: map[component.ID]component.Config{
			component.NewID(nopType): exporterFactory.CreateDefaultConfig(),
		},
		ExportersFactories: map[component.Type]exporter.Factory{
			nopType: exporterFactory,
		},
		ConnectorsConfigs: map[component.ID]component.Config{
			connID: connectorFactory.CreateDefaultConfig(),
		},
		ConnectorsFactories: map[component.Type]connector.Factory{
			nopType: connectorFactory,
		},
		ExtensionsConfigs: map[component.ID]component.Config{
			component.NewID(component.MustNewType("watcher")): &extensionConfig{eventsReceived},
		},
		ExtensionsFactories: map[component.Type]extension.Factory{
			component.MustNewType("watcher"): newExtensionFactory(),
		},
		TelemetryFactory: otelconftelemetry.NewFactory(),
	}
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: "otelcoltest"}

	cfg := service.Config{
		Telemetry: &otelconftelemetry.Config{
			Logs: otelconftelemetry.LogsConfig{
				Level:       zapcore.InfoLevel,
				Development: false,
				Encoding:    "console",
				Sampling: &otelconftelemetry.LogsSamplingConfig{
					Enabled:    true,
					Tick:       10 * time.Second,
					Initial:    100,
					Thereafter: 100,
				},
				OutputPaths:       []string{"stderr"},
				ErrorOutputPaths:  []string{"stderr"},
				DisableCaller:     false,
				DisableStacktrace: false,
				InitialFields:     map[string]any(nil),
			},
			Metrics: otelconftelemetry.MetricsConfig{
				Level: configtelemetry.LevelNone,
			},
		},
		Pipelines: pipelines.Config{
			pipeline.NewID(pipeline.SignalTraces): {
				Receivers: []component.ID{component.NewID(component.MustNewType("test"))},
				Exporters: []component.ID{component.NewID(nopType)},
			},
			pipeline.NewID(pipeline.SignalMetrics): {
				Receivers: []component.ID{component.NewID(component.MustNewType("test"))},
				Exporters: []component.ID{component.NewID(nopType)},
			},
		},
		Extensions: extensions.Config{component.NewID(component.MustNewType("watcher"))},
	}

	s, err := service.New(context.Background(), set, cfg)
	require.NoError(t, err)

	wg.Add(1)
	err = s.Start(context.Background())
	require.NoError(t, err)
	wg.Wait()
	err = s.Shutdown(context.Background())
	require.NoError(t, err)

	require.Len(t, eventsReceived, 2)

	for instanceID, events := range eventsReceived {
		pipelineIDs := ""
		instanceID.AllPipelineIDs(func(id pipeline.ID) bool {
			pipelineIDs += id.String() + ","
			return true
		})

		t.Logf("checking errors for %v - %v - %v", pipelineIDs, instanceID.Kind().String(), instanceID.ComponentID().String())

		var expectedEvents []*componentstatus.Event
		// The StatusOk is not guaranteed to be in the slice, set it according to the number of captured states
		assert.True(t, len(events) == 4 || len(events) == 5)
		receiverTestAttrs := pcommon.NewMap()
		receiverTestAttrs.PutStr("scraper", "test")
		if len(events) == 4 {
			expectedEvents = []*componentstatus.Event{
				componentstatus.NewEvent(componentstatus.StatusStarting),
				componentstatus.NewEvent(componentstatus.StatusRecoverableError, componentstatus.WithAttributes(receiverTestAttrs)),
				componentstatus.NewEvent(componentstatus.StatusStopping),
				componentstatus.NewEvent(componentstatus.StatusStopped),
			}
		} else {
			expectedEvents = []*componentstatus.Event{
				componentstatus.NewEvent(componentstatus.StatusStarting),
				componentstatus.NewEvent(componentstatus.StatusRecoverableError, componentstatus.WithAttributes(receiverTestAttrs)),
				componentstatus.NewEvent(componentstatus.StatusOK),
				componentstatus.NewEvent(componentstatus.StatusStopping),
				componentstatus.NewEvent(componentstatus.StatusStopped),
			}
		}

		var eventStr strings.Builder
		for i, e := range events {
			fmt.Fprintf(&eventStr, "%v,", e.Status())
			assert.Equal(t, expectedEvents[i].Status(), e.Status())
		}
		t.Logf("events received: %v", eventStr.String())
	}
}

func newReceiverFactory() receiver.Factory {
	return receiver.NewFactory(
		component.MustNewType("test"),
		createDefaultReceiverConfig,
		receiver.WithTraces(createTraces, component.StabilityLevelStable),
		receiver.WithMetrics(createMetrics, component.StabilityLevelStable),
	)
}

type testReceiver struct{}

func (t *testReceiver) Start(_ context.Context, host component.Host) error {
	scraperAttrs := pcommon.NewMap()
	scraperAttrs.PutStr("scraper", "test")
	componentstatus.ReportStatus(host, componentstatus.NewEvent(
		componentstatus.StatusRecoverableError,
		componentstatus.WithError(errors.New("test recoverable error")),
		componentstatus.WithAttributes(scraperAttrs),
	))

	go func() {
		componentstatus.ReportStatus(host, componentstatus.NewEvent(componentstatus.StatusOK))
		wg.Done()
	}()
	return nil
}

func (t *testReceiver) Shutdown(_ context.Context) error {
	return nil
}

type receiverConfig struct{}

func createDefaultReceiverConfig() component.Config {
	return &receiverConfig{}
}

func createTraces(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	_ consumer.Traces,
) (receiver.Traces, error) {
	oCfg := cfg.(*receiverConfig)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*testReceiver, error) {
			return &testReceiver{}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func createMetrics(
	_ context.Context,
	_ receiver.Settings,
	cfg component.Config,
	_ consumer.Metrics,
) (receiver.Metrics, error) {
	oCfg := cfg.(*receiverConfig)
	r, err := receivers.LoadOrStore(
		oCfg,
		func() (*testReceiver, error) {
			return &testReceiver{}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

var receivers = sharedcomponent.NewMap[*receiverConfig, *testReceiver]()

func newExtensionFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType("watcher"),
		createDefaultExtensionConfig,
		create,
		component.StabilityLevelStable,
	)
}

func create(_ context.Context, _ extension.Settings, cfg component.Config) (extension.Extension, error) {
	oCfg := cfg.(*extensionConfig)
	return &testExtension{
		eventsReceived: oCfg.eventsReceived,
	}, nil
}

type testExtension struct {
	eventsReceived map[*componentstatus.InstanceID][]*componentstatus.Event
}

type extensionConfig struct {
	eventsReceived map[*componentstatus.InstanceID][]*componentstatus.Event
}

func createDefaultExtensionConfig() component.Config {
	return &extensionConfig{}
}

// Start implements the component.Component interface.
func (t *testExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown implements the component.Component interface.
func (t *testExtension) Shutdown(_ context.Context) error {
	return nil
}

// ComponentStatusChanged implements the extension.StatusWatcher interface.
func (t *testExtension) ComponentStatusChanged(
	source *componentstatus.InstanceID,
	event *componentstatus.Event,
) {
	if source.ComponentID() == component.NewID(component.MustNewType("test")) {
		t.eventsReceived[source] = append(t.eventsReceived[source], event)
	}
}

// NotifyConfig implements the extensioncapabilities.ConfigWatcher interface.
func (t *testExtension) NotifyConfig(_ context.Context, _ *confmap.Conf) error {
	return nil
}

// Ready implements the extensioncapabilities.PipelineWatcher interface.
func (t *testExtension) Ready() error {
	return nil
}

// NotReady implements the extensioncapabilities.PipelineWatcher interface.
func (t *testExtension) NotReady() error {
	return nil
}
