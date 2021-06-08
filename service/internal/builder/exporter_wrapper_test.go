package builder

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
)

type exampleExporter struct {
	startErr          error
	shutdownErr       error
	consumeLogsErr    error
	consumeMetricsErr error
	consumeTracesErr  error
	receivedLogs      pdata.Logs
	receivedMetrics   pdata.Metrics
	receivedTraces    pdata.Traces
	cap               consumer.Capabilities
}

type reloadableExporter struct {
	exampleExporter
	reloadErr error
}

func (exp *reloadableExporter) Relaod(host component.Host, ctx context.Context, cfg interface{}) error {
	return exp.reloadErr
}

var defaultExampleExporter *exampleExporter = &exampleExporter{}

const expType = "example"

var exampleExporterFactory = exporterhelper.NewFactory(
	expType,
	createDefaultConfig,
	exporterhelper.WithTraces(createTracesExporter),
	exporterhelper.WithMetrics(createMetricsExporter),
	exporterhelper.WithLogs(createLogsExporter))

func (exp *exampleExporter) Start(ctx context.Context, host component.Host) error {
	time.Sleep(time.Second)
	return exp.startErr
}

func (exp *exampleExporter) Shutdown(ctx context.Context) error {
	time.Sleep(time.Second)
	return exp.shutdownErr
}

func (exp *exampleExporter) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	exp.receivedLogs = ld
	return exp.consumeLogsErr
}

func (exp *exampleExporter) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	exp.receivedTraces = td
	return exp.consumeTracesErr
}

func (exp *exampleExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	exp.receivedMetrics = md
	return exp.consumeMetricsErr
}

func (exp *exampleExporter) Capabilities() consumer.Capabilities {
	return exp.cap
}

func createDefaultConfig() config.Exporter {
	setting := config.NewExporterSettings(config.NewIDWithName("example", "1"))
	return &setting
}

func createTracesExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ config.Exporter,
) (component.TracesExporter, error) {
	return &exampleExporter{}, nil
}

func createMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ config.Exporter,
) (component.MetricsExporter, error) {
	return &exampleExporter{}, nil
}

func createLogsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	_ config.Exporter,
) (component.LogsExporter, error) {
	return &exampleExporter{}, nil
}

func TestExporterWrapperConsumeAndNormalReload(t *testing.T) {
	exp := &exampleExporter{}
	logWrapper := &exporterWrapper{config.LogsDataType, nil, nil, exp, exp, config.NewIDWithName("example", "1"), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory}

	// Test Logs
	ld := pdata.NewLogs()
	err := logWrapper.ConsumeLogs(context.Background(), ld)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedLogs, ld)
	mockConsumeLogErr := errors.New("mock consume log error")
	exp.consumeLogsErr = mockConsumeLogErr
	err = logWrapper.ConsumeLogs(context.Background(), pdata.NewLogs())
	assert.Equal(t, err, mockConsumeLogErr)
	err = logWrapper.ConsumeMetrics(context.Background(), pdata.NewMetrics())
	assert.Equal(t, err, errDataTypeNotSupported)
	err = logWrapper.ConsumeTraces(context.Background(), pdata.NewTraces())
	assert.Equal(t, err, errDataTypeNotSupported)
	exp.consumeLogsErr = nil
	stop := false
	switchExp := false
	go func() {
		for !stop {
			ld := pdata.NewLogs()
			err := logWrapper.ConsumeLogs(context.Background(), ld)
			assert.NoError(t, err)
			if logWrapper.lc == exp {
				assert.Equal(t, exp.receivedLogs, ld)
			} else {
				switchExp = true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	go func() {
		err := logWrapper.Relaod(componenttest.NewNopHost(), context.Background(), createDefaultConfig())
		assert.NoError(t, err)
	}()

	assert.False(t, switchExp)
	time.Sleep(1090 * time.Millisecond)
	stop = true
	assert.True(t, switchExp)
	assert.False(t, logWrapper.lc == exp)
	assert.True(t, logWrapper.lc == logWrapper.exporter.(consumer.Logs))
	err = logWrapper.ConsumeLogs(context.Background(), ld)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedLogs, ld)
	invalidIdConfig := config.NewExporterSettings(config.NewIDWithName("example", "2"))
	err = logWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidIdConfig)
	assert.Error(t, err)
	invalidTypeConfig := config.NewIDWithName("example", "1")
	err = logWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidTypeConfig)
	assert.Error(t, err)

	// Test Metrics
	metricWrapper := &exporterWrapper{config.MetricsDataType, exp, nil, nil, exp, config.NewIDWithName("example", "1"), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory}
	md := pdata.NewMetrics()
	err = metricWrapper.ConsumeMetrics(context.Background(), md)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedMetrics, md)
	mockConsumeMetricErr := errors.New("mock consume metric error")
	exp.consumeMetricsErr = mockConsumeMetricErr
	err = metricWrapper.ConsumeMetrics(context.Background(), md)
	assert.Equal(t, err, mockConsumeMetricErr)
	err = metricWrapper.ConsumeLogs(context.Background(), pdata.NewLogs())
	assert.Equal(t, err, errDataTypeNotSupported)
	err = metricWrapper.ConsumeTraces(context.Background(), pdata.NewTraces())
	assert.Equal(t, err, errDataTypeNotSupported)
	exp.consumeMetricsErr = nil
	stop = false
	switchExp = false
	go func() {
		for !stop {
			md := pdata.NewMetrics()
			err := metricWrapper.ConsumeMetrics(context.Background(), md)
			assert.NoError(t, err)
			if metricWrapper.mc == exp {
				assert.Equal(t, exp.receivedMetrics, md)
			} else {
				switchExp = true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	go func() {
		err := metricWrapper.Relaod(componenttest.NewNopHost(), context.Background(), createDefaultConfig())
		assert.NoError(t, err)
	}()

	assert.False(t, switchExp)
	time.Sleep(1090 * time.Millisecond)
	stop = true
	assert.True(t, switchExp)
	assert.False(t, metricWrapper.mc == exp)
	assert.True(t, metricWrapper.mc == metricWrapper.exporter.(consumer.Metrics))
	err = metricWrapper.ConsumeMetrics(context.Background(), md)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedMetrics, md)
	invalidIdConfig = config.NewExporterSettings(config.NewIDWithName("example", "2"))
	err = metricWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidIdConfig)
	assert.Error(t, err)
	invalidTypeConfig = config.NewIDWithName("example", "1")
	err = metricWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidTypeConfig)
	assert.Error(t, err)

	// Test Traces
	traceWrapper := &exporterWrapper{config.TracesDataType, nil, exp, nil, exp, config.NewIDWithName("example", "1"), zap.NewNop(), component.DefaultBuildInfo(), exampleExporterFactory}
	td := pdata.NewTraces()
	err = traceWrapper.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedTraces, td)
	mockConsumeTraceErr := errors.New("mock consume trace error")
	exp.consumeTracesErr = mockConsumeTraceErr
	err = traceWrapper.ConsumeTraces(context.Background(), td)
	assert.Equal(t, err, mockConsumeTraceErr)
	err = traceWrapper.ConsumeLogs(context.Background(), pdata.NewLogs())
	assert.Equal(t, err, errDataTypeNotSupported)
	err = traceWrapper.ConsumeMetrics(context.Background(), pdata.NewMetrics())
	assert.Equal(t, err, errDataTypeNotSupported)
	exp.consumeTracesErr = nil
	stop = false
	switchExp = false
	go func() {
		for !stop {
			td := pdata.NewTraces()
			err := traceWrapper.ConsumeTraces(context.Background(), td)
			assert.NoError(t, err)
			if traceWrapper.tc == exp {
				assert.Equal(t, exp.receivedTraces, td)
			} else {
				switchExp = true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	go func() {
		err := traceWrapper.Relaod(componenttest.NewNopHost(), context.Background(), createDefaultConfig())
		assert.NoError(t, err)
	}()

	assert.False(t, switchExp)
	time.Sleep(1090 * time.Millisecond)
	stop = true
	assert.True(t, switchExp)
	assert.False(t, traceWrapper.tc == exp)
	assert.True(t, traceWrapper.tc == traceWrapper.exporter.(consumer.Traces))
	err = traceWrapper.ConsumeTraces(context.Background(), td)
	assert.NoError(t, err)
	assert.Equal(t, exp.receivedTraces, td)
	invalidIdConfig = config.NewExporterSettings(config.NewIDWithName("example", "2"))
	err = traceWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidIdConfig)
	assert.Error(t, err)
	invalidTypeConfig = config.NewIDWithName("example", "1")
	err = traceWrapper.Relaod(componenttest.NewNopHost(), context.Background(), &invalidTypeConfig)
	assert.Error(t, err)
}
