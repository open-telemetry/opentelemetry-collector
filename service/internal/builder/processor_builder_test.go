package builder

import (
	"context"
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
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const expProcType = "example"
const rldExpProcType = "rldexmaple"

var exampleProcessorFactory = processorhelper.NewFactory(
	expProcType,
	createDefaultProcConfig,
	processorhelper.WithTraces(createTracesProcessor),
	processorhelper.WithMetrics(createMetricsProcessor),
	processorhelper.WithLogs(createLogsProcessor))

func createDefaultProcConfig() config.Processor {
	setting := config.NewProcessorSettings(config.NewID(expProcType))
	return &setting
}

func createTracesProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	return &exampleProcessor{Traces: nextConsumer}, nil
}

func createMetricsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Metrics) (component.MetricsProcessor, error) {
	return &exampleProcessor{Metrics: nextConsumer}, nil
}

func createLogsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Logs) (component.LogsProcessor, error) {
	return &exampleProcessor{Logs: nextConsumer}, nil
}

var rldExampleProcessorFactory = processorhelper.NewFactory(
	rldExpProcType,
	createRldDefaultProcConfig,
	processorhelper.WithTraces(createRldTracesProcessor),
	processorhelper.WithMetrics(createRldMetricsProcessor),
	processorhelper.WithLogs(createRldLogsProcessor))

func createRldDefaultProcConfig() config.Processor {
	setting := config.NewProcessorSettings(config.NewID(rldExpProcType))
	return &setting
}

func createRldTracesProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	return &rldExampleProcessor{exampleProcessor: exampleProcessor{Traces: nextConsumer}}, nil
}

func createRldMetricsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Metrics) (component.MetricsProcessor, error) {
	return &rldExampleProcessor{exampleProcessor: exampleProcessor{Metrics: nextConsumer}}, nil
}

func createRldLogsProcessor(_ context.Context, _ component.ProcessorCreateSettings, _ config.Processor, nextConsumer consumer.Logs) (component.LogsProcessor, error) {
	return &rldExampleProcessor{exampleProcessor: exampleProcessor{Logs: nextConsumer}}, nil
}

type exampleProcessor struct {
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

func (ep *exampleProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (ep *exampleProcessor) Shutdown(_ context.Context) error {
	return nil
}

func (ep *exampleProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type rldExampleProcessor struct {
	exampleProcessor
	reloadErr   error
	reloadCount int
}

func (exp *rldExampleProcessor) Relaod(host component.Host, ctx context.Context, cfg interface{}) error {
	time.Sleep(time.Second)
	exp.reloadCount += 1
	return exp.reloadErr
}

func TestProcessorsReload(t *testing.T) {
	factories, err := testcomponents.ExampleComponents()
	factories.Processors[exampleProcessorFactory.Type()] = exampleProcessorFactory
	factories.Processors[rldExampleProcessorFactory.Type()] = rldExampleProcessorFactory
	assert.NoError(t, err)
	cfg, err := configtest.LoadConfigAndValidate("testdata/reload_pipelines_builder.yaml", factories)
	// Load the config
	require.Nil(t, err)

	// BuildProcessors the pipeline
	allExporters, err := BuildExporters(zap.NewNop(), component.DefaultBuildInfo(), cfg, factories.Exporters)
	assert.NoError(t, err)
	pipelineProcessors, err := BuildPipelines(zap.NewNop(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)

	assert.NoError(t, err)
	require.NotNil(t, pipelineProcessors)

	assert.NoError(t, pipelineProcessors.StartProcessors(context.Background(), componenttest.NewNopHost()))

	oldExpProc := pipelineProcessors["traces"].btProcs[0].tc.(*exampleProcessor)
	err = pipelineProcessors.ReloadProcessors(context.Background(), componenttest.NewNopHost(), component.DefaultBuildInfo(), cfg, allExporters, factories.Processors)
	assert.NoError(t, err)
	// Ensure pipeline has its fields correctly populated.
	btProc := pipelineProcessors["traces"].btProcs[1]
	assert.Equal(t, btProc.tc.(*rldExampleProcessor).reloadCount, 1)
	assert.False(t, oldExpProc == pipelineProcessors["traces"].btProcs[0].tc)

	for pipelineName, pipeline := range cfg.Service.Pipelines {
		processor := pipelineProcessors[pipelineName]
		require.NotNil(t, processor)
		if pipeline.InputType == config.TracesDataType {
			assert.NotNil(t, processor.firstTC)
			assert.Nil(t, processor.firstMC)
		}

		if pipeline.InputType == config.MetricsDataType {
			assert.NotNil(t, processor.firstMC)
			assert.Nil(t, processor.firstTC)
		}

		// Compose the list of created exporters.
		var exporters []*builtExporter
		for _, name := range pipeline.Exporters {
			// Ensure exporter is created.
			exp := allExporters[name]
			require.NotNil(t, exp)
			exporters = append(exporters, exp)
		}

		// Send TraceData via processor and verify that all exporters of the pipeline receive it.

		if pipeline.InputType == config.TracesDataType {
			var exporterConsumers []*testcomponents.ExampleExporterConsumer
			for _, exporter := range exporters {
				expConsumer := exporter.getTracesExporter().(*exporterWrapper).tc.(*testcomponents.ExampleExporterConsumer)
				exporterConsumers = append(exporterConsumers, expConsumer)
				require.Equal(t, len(expConsumer.Traces), 0)
			}
			td := testdata.GenerateTracesOneSpan()
			require.NoError(t, processor.firstTC.(consumer.Traces).ConsumeTraces(context.Background(), td))

			// Now verify received data.
			for _, expConsumer := range exporterConsumers {
				// Check that the trace is received by exporter.
				require.Equal(t, 1, len(expConsumer.Traces))

				// Verify that span is successfully delivered.
				assert.EqualValues(t, td, expConsumer.Traces[0])
			}
		}

		if pipeline.InputType == config.MetricsDataType {
			var exporterConsumers []*testcomponents.ExampleExporterConsumer
			for _, exporter := range exporters {
				expConsumer := exporter.getMetricExporter().(*exporterWrapper).mc.(*testcomponents.ExampleExporterConsumer)
				exporterConsumers = append(exporterConsumers, expConsumer)
				require.Equal(t, len(expConsumer.Traces), 0)
			}
			md := testdata.GenerateMetricsOneMetric()
			require.NoError(t, processor.firstMC.(consumer.Metrics).ConsumeMetrics(context.Background(), md))

			// Now verify received data.
			for _, expConsumer := range exporterConsumers {
				// Check that the trace is received by exporter.
				require.Equal(t, 1, len(expConsumer.Metrics))

				// Verify that span is successfully delivered.
				assert.EqualValues(t, md, expConsumer.Metrics[0])
			}
		}
	}

	err = pipelineProcessors.ShutdownProcessors(context.Background())
	assert.NoError(t, err)
}
