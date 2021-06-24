package builder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const expRcvType = "example"
const rldExpRcvType = "rldexample"

var receivers = sharedcomponent.NewSharedComponents()

var exampleReceiverFactory = receiverhelper.NewFactory(
	expRcvType,
	createDefaultRcvConfig,
	receiverhelper.WithTraces(createTracesReceiver),
	receiverhelper.WithMetrics(createMetricsReceiver),
	receiverhelper.WithLogs(createLogsReceiver))

func createDefaultRcvConfig() config.Receiver {
	setting := config.NewReceiverSettings(config.NewID(expRcvType))
	return &setting
}

func createTracesReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Traces) (component.TracesReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &exampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*exampleRcv); ok {
		rcv.Traces = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

func createMetricsReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Metrics) (component.MetricsReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &exampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*exampleRcv); ok {
		rcv.Metrics = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

func createLogsReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Logs) (component.LogsReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &exampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*exampleRcv); ok {
		rcv.Logs = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

var rldExampleReceiverFactory = receiverhelper.NewFactory(
	rldExpRcvType,
	createRldDefaultRcvConfig,
	receiverhelper.WithTraces(createRldTracesReceiver),
	receiverhelper.WithMetrics(createRldMetricsReceiver),
	receiverhelper.WithLogs(createRldLogsReceiver))

func createRldDefaultRcvConfig() config.Receiver {
	setting := config.NewReceiverSettings(config.NewID(rldExpRcvType))
	return &setting
}

func createRldTracesReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Traces) (component.TracesReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &rldExampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*rldExampleRcv); ok {
		rcv.Traces = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

func createRldMetricsReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Metrics) (component.MetricsReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &rldExampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*rldExampleRcv); ok {
		rcv.Metrics = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

func createRldLogsReceiver(_ context.Context, _ component.ReceiverCreateParams, cfg config.Receiver, nextConsumer consumer.Logs) (component.LogsReceiver, error) {
	r := receivers.GetOrAdd(cfg, func() component.Component {
		return &rldExampleRcv{}
	})
	if rcv, ok := r.Unwrap().(*rldExampleRcv); ok {
		rcv.Logs = nextConsumer
		return rcv, nil
	}
	return nil, errors.New("unknown error")
}

type exampleRcv struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

type rldExampleRcv struct {
	exampleRcv
	reloadCount int
}

func (rcv *exampleRcv) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (rcv *exampleRcv) Shutdown(_ context.Context) error {
	return nil
}

func (rcv *exampleRcv) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (rcv *rldExampleRcv) Reload(host component.Host, ctx context.Context, cfg interface{}) error {
	time.Sleep(time.Second)
	rcv.reloadCount += 1
	return nil
}

func TestReloadReceivers(t *testing.T) {
	tests := []testCase{
		{
			name:        "one-exporter",
			receiverID:  config.NewID("example"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter")},
			hasTraces:   true,
			hasMetrics:  true,
		},
		{
			name:        "multi-exporter",
			receiverID:  config.NewIDWithName("example", "2"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},
			hasTraces:   true,
		},
		{
			name:        "multi-metrics-receiver",
			receiverID:  config.NewIDWithName("rldexample", "3"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},
			hasTraces:   false,
			hasMetrics:  true,
		},
		{
			name:        "multi-receiver-multi-exporter",
			receiverID:  config.NewIDWithName("rldexample", "multi"),
			exporterIDs: []config.ComponentID{config.NewID("exampleexporter"), config.NewIDWithName("exampleexporter", "2")},

			// Check pipelines_builder.yaml to understand this case.
			// We have 2 pipelines, one exporting to one exporter, the other
			// exporting to both exporters, so we expect a duplication on
			// one of the exporters, but not on the other.
			spanDuplicationByExporter: map[config.ComponentID]int{
				config.NewID("exampleexporter"): 2, config.NewIDWithName("exampleexporter", "2"): 1,
			},
			hasTraces: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testReloadReceivers(t, test)
		})
	}
}

func testReloadReceivers(t *testing.T, test testCase) {
	factories, err := testcomponents.ExampleComponents()
	factories.Receivers[exampleReceiverFactory.Type()] = exampleReceiverFactory
	factories.Receivers[rldExampleReceiverFactory.Type()] = rldExampleReceiverFactory
	assert.NoError(t, err)

	cfg, err := configtest.LoadConfigFile(t, "testdata/reload_receiver.yaml", factories)
	require.NoError(t, err)

	// Build the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
	assert.NoError(t, err)
	receivers, err := BuildReceivers(zap.NewNop(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers)
	assert.NoError(t, err)

	err = allExporters.StartAll(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)
	err = pipelineProcessors.StartProcessors(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)
	err = receivers.StartAll(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)

	oldRcvs := make(map[config.ComponentID]component.Receiver)
	for id, rvc := range receivers {
		oldRcvs[id] = rvc.receiver
	}

	cfg, err = configtest.LoadConfigFile(t, "testdata/reload_receiver.yaml", factories)
	require.NoError(t, err)
	err = allExporters.ReloadExporters(context.Background(), zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters, componenttest.NewNopHost())
	assert.NoError(t, err)
	err = pipelineProcessors.ReloadProcessors(context.Background(), componenttest.NewNopHost(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
	assert.NoError(t, err)
	err = receivers.ReloadReceivers(context.Background(), zap.NewNop(), component.DefaultBuildInfo(), cfg, pipelineProcessors, factories.Receivers, componenttest.NewNopHost())
	assert.NoError(t, err)
	require.NotNil(t, receivers)

	for id, rcv := range receivers {
		if id.Type() == expRcvType {
			assert.False(t, rcv.receiver == oldRcvs[id])
		}

		if id.Type() == rldExpRcvType {
			assert.True(t, rcv.receiver == oldRcvs[id])
			assert.Equal(t, rcv.receiver.(*rldExampleRcv).reloadCount, 1)
		}
	}

	receiver := receivers[test.receiverID]

	// Ensure receiver has its fields correctly populated.
	require.NotNil(t, receiver)
	assert.NotNil(t, receiver.receiver)

	// Compose the list of created exporters.
	var exporters []*builtExporter
	for _, name := range test.exporterIDs {
		// Ensure exporter is created.
		exp := allExporters[name]
		require.NotNil(t, exp)
		exporters = append(exporters, exp)
	}

	// Send TraceData via receiver and verify that all exporters of the pipeline receive it.

	// First check that there are no traces in the exporters yet.
	for _, exporter := range exporters {
		consumer := exporter.getTracesExporter().(*exporterWrapper).tc.(*testcomponents.ExampleExporterConsumer)
		require.Equal(t, len(consumer.Traces), 0)
		require.Equal(t, len(consumer.Metrics), 0)
	}

	td := testdata.GenerateTracesOneSpan()
	if test.hasTraces {
		traceProducer := receiver.receiver.(consumer.Traces)
		assert.NoError(t, traceProducer.ConsumeTraces(context.Background(), td))
	}

	md := testdata.GenerateMetricsOneMetric()
	if test.hasMetrics {
		metricsProducer := receiver.receiver.(consumer.Metrics)
		assert.NoError(t, metricsProducer.ConsumeMetrics(context.Background(), md))
	}

	// Now verify received data.
	for _, name := range test.exporterIDs {
		// Check that the data is received by exporter.
		exporter := allExporters[name]

		// Validate traces.
		if test.hasTraces {
			var spanDuplicationCount int
			if test.spanDuplicationByExporter != nil {
				spanDuplicationCount = test.spanDuplicationByExporter[name]
			} else {
				spanDuplicationCount = 1
			}

			traceConsumer := exporter.getTracesExporter().(*exporterWrapper).tc.(*testcomponents.ExampleExporterConsumer)
			require.Equal(t, spanDuplicationCount, len(traceConsumer.Traces))

			for i := 0; i < spanDuplicationCount; i++ {
				assert.EqualValues(t, td, traceConsumer.Traces[i])
			}
		}

		// Validate metrics.
		if test.hasMetrics {
			metricsConsumer := exporter.getMetricExporter().(*exporterWrapper).mc.(*testcomponents.ExampleExporterConsumer)
			require.Equal(t, 1, len(metricsConsumer.Metrics))
			assert.EqualValues(t, md, metricsConsumer.Metrics[0])
		}
	}
}
