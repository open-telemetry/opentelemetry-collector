package builder

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const expRcvType = "example"
const rldExpRcvType = "rldexample"

var exampleReceiverFactory = receiverhelper.NewFactory(
	expProcType,
	createDefaultRcvConfig,
	receiverhelper.WithTraces(createTracesReceiver),
	receiverhelper.WithMetrics(createMetricsReceiver),
	receiverhelper.WithLogs(createLogsReceiver))

func createDefaultRcvConfig() config.Receiver {
	setting := config.NewReceiverSettings(config.NewID(expRcvType))
	return &setting
}

func createTracesReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Traces) (component.TracesReceiver, error) {
	return &exampleRcv{Traces: nextConsumer}, nil
}

func createMetricsReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Metrics) (component.MetricsReceiver, error) {
	return &exampleRcv{Metrics: nextConsumer}, nil
}

func createLogsReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Logs) (component.LogsReceiver, error) {
	return &exampleRcv{Logs: nextConsumer}, nil
}

var rldExampleReceiverFactory = receiverhelper.NewFactory(
	rldExpProcType,
	createRldDefaultRcvConfig,
	receiverhelper.WithTraces(createRldTracesReceiver),
	receiverhelper.WithMetrics(createRldMetricsReceiver),
	receiverhelper.WithLogs(createRldLogsReceiver))

func createRldDefaultRcvConfig() config.Receiver {
	setting := config.NewReceiverSettings(config.NewID(rldExpRcvType))
	return &setting
}

func createRldTracesReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Traces) (component.TracesReceiver, error) {
	return &rldExampleRcv{exampleRcv: exampleRcv{Traces: nextConsumer}}, nil
}

func createRldMetricsReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Metrics) (component.MetricsReceiver, error) {
	return &rldExampleRcv{exampleRcv: exampleRcv{Metrics: nextConsumer}}, nil
}

func createRldLogsReceiver(_ context.Context, _ component.ReceiverCreateParams, _ config.Receiver, nextConsumer consumer.Logs) (component.LogsReceiver, error) {
	return &rldExampleRcv{exampleRcv: exampleRcv{Logs: nextConsumer}}, nil
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

func (rcv *rldExampleRcv) Relaod(host component.Host, ctx context.Context, cfg interface{}) error {
	time.Sleep(time.Second)
	rcv.reloadCount += 1
	return nil
}
