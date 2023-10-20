// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exportertest

import (
	"context"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func (bor *testMockReceiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (bor *testMockReceiver) Shutdown(_ context.Context) error {
	return nil
}

func newExampleFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithLogs(CreateLogsExporter, component.StabilityLevelBeta),
		exporter.WithTraces(CreateTracesExporter, component.StabilityLevelBeta),
		exporter.WithMetrics(CreateMetricsExporter, component.StabilityLevelBeta),
	)
}

func CreateLogsExporter(_ context.Context, _ exporter.CreateSettings, cfg component.Config) (exporter.Logs, error) {
	set := exporter.CreateSettings{
		ID: component.NewIDWithName(component.DataTypeLogs, "test-exporter"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:                zap.NewNop(),
			TracerProvider:        trace.NewNoopTracerProvider(),
			MeterProvider:         noop.NewMeterProvider(),
			MetricsLevel:          0,
			Resource:              pcommon.Resource{},
			ReportComponentStatus: nil,
		},
		BuildInfo: component.BuildInfo{Version: "0.0.0"},
	}
	expConfig := cfg.(*ConnectionConfig)
	le, err := exporterhelper.NewLogsExporter(
		context.Background(),
		set,
		cfg,
		pushLogs,
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(expConfig.RetrySettings),
		exporterhelper.WithQueue(expConfig.QueueSettings))
	return le, err
}

func CreateTracesExporter(_ context.Context, _ exporter.CreateSettings, cfg component.Config) (exporter.Traces, error) {
	set := exporter.CreateSettings{
		ID: component.NewIDWithName(component.DataTypeTraces, "test-exporter"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:                zap.NewNop(),
			TracerProvider:        trace.NewNoopTracerProvider(),
			MeterProvider:         noop.NewMeterProvider(),
			MetricsLevel:          0,
			Resource:              pcommon.Resource{},
			ReportComponentStatus: nil,
		},
		BuildInfo: component.BuildInfo{Version: "0.0.0"},
	}
	expConfig := cfg.(*ConnectionConfig)
	le, err := exporterhelper.NewTracesExporter(
		context.Background(),
		set,
		cfg,
		pushTraces,
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(expConfig.RetrySettings),
		exporterhelper.WithQueue(expConfig.QueueSettings))
	return le, err
}

func pushTraces(_ context.Context, _ ptrace.Traces) error {
	return pushAny()
}

func CreateMetricsExporter(_ context.Context, _ exporter.CreateSettings, cfg component.Config) (exporter.Metrics, error) {
	set := exporter.CreateSettings{
		ID: component.NewIDWithName(component.DataTypeMetrics, "test-exporter"),
		TelemetrySettings: component.TelemetrySettings{
			Logger:                zap.NewNop(),
			TracerProvider:        trace.NewNoopTracerProvider(),
			MeterProvider:         noop.NewMeterProvider(),
			MetricsLevel:          0,
			Resource:              pcommon.Resource{},
			ReportComponentStatus: nil,
		},
		BuildInfo: component.BuildInfo{Version: "0.0.0"},
	}
	expConfig := cfg.(*ConnectionConfig)
	le, err := exporterhelper.NewMetricsExporter(
		context.Background(),
		set,
		cfg,
		pushMetrics,
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(expConfig.RetrySettings),
		exporterhelper.WithQueue(expConfig.QueueSettings))
	return le, err
}

func pushAny() error {
	joinedRequestCounter.total++
	generatedError := fakeMockConsumer.exportErrorFunction()
	if generatedError != nil {
		if consumererror.IsPermanent(generatedError) {
			joinedRequestCounter.error.permanent++
		} else {
			joinedRequestCounter.error.nonpermanent++
		}
		return generatedError
	}
	joinedRequestCounter.success++
	return nil
}

func pushMetrics(_ context.Context, _ pmetric.Metrics) error {
	return pushAny()
}

func pushLogs(_ context.Context, _ plog.Logs) error {
	return pushAny()
}

func createDefaultConfig() component.Config {
	return &ConnectionConfig{
		TimeoutSettings: exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:   exporterhelper.NewDefaultQueueSettings(),
	}
}

func newTestRetrySettings() exporterhelper.RetrySettings {
	return exporterhelper.RetrySettings{
		Enabled: true,
		// interval is short for the test purposes
		InitialInterval:     10 * time.Millisecond,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          1.1,
		MaxInterval:         10 * time.Second,
		MaxElapsedTime:      1 * time.Minute,
	}
}

type ConnectionConfig struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`
}

func testConfig() *ConnectionConfig {
	return &ConnectionConfig{
		TimeoutSettings: exporterhelper.TimeoutSettings{},
		QueueSettings:   exporterhelper.QueueSettings{Enabled: false},
		RetrySettings:   newTestRetrySettings(),
	}
}

type testMockReceiver struct {
	mockConsumer *MockConsumer
}

func newOTLPDataReceiver(mockConsumer *MockConsumer) *testMockReceiver {
	return &testMockReceiver{mockConsumer: mockConsumer}
}

var joinedRequestCounter = newRequestCounter()

var fakeMockConsumer = &MockConsumer{reqCounter: joinedRequestCounter}

// Define a function that matches the MockReceiverFactory signature
func createMockOtlpReceiver(mockConsumer *MockConsumer) component.Component {
	rcv := newOTLPDataReceiver(mockConsumer)
	joinedRequestCounter = newRequestCounter()
	fakeMockConsumer.exportErrorFunction = mockConsumer.exportErrorFunction
	mockConsumer.reqCounter = joinedRequestCounter
	err := rcv.Start(context.Background(), nil)
	if err != nil {
		return nil
	}
	return rcv
}

func TestCheckConsumeContractLogs(t *testing.T) {

	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              newExampleFactory(),    // Replace with your exporter factory
		DataType:             component.DataTypeLogs, // Change to the appropriate data type
		Config:               testConfig(),
		NumberOfTestElements: 10, // Number of test elements you want to send
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	CheckConsumeContract(params)
}

func TestCheckConsumeContractMetrics(t *testing.T) {

	// Create a CheckConsumeContractParams
	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              newExampleFactory(),       // Replace with your exporter factory
		DataType:             component.DataTypeMetrics, // Change to the appropriate data type
		Config:               testConfig(),
		NumberOfTestElements: 10, // Number of test elements you want to send
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	CheckConsumeContract(params)
}

func TestCheckConsumeContractTraces(t *testing.T) {

	params := CheckConsumeContractParams{
		T:                    t,
		Factory:              newExampleFactory(),      // Replace with your exporter factory
		DataType:             component.DataTypeTraces, // Change to the appropriate data type
		Config:               testConfig(),
		NumberOfTestElements: 10, // Number of test elements you want to send
		MockReceiverFactory:  createMockOtlpReceiver,
	}

	CheckConsumeContract(params)
}
