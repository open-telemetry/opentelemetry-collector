package builder

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type exporterWrapper struct {
	inputType config.DataType
	mc        consumer.Metrics
	tc        consumer.Traces
	lc        consumer.Logs
	exporter  component.Exporter
}

func (wrapper *exporterWrapper) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	return wrapper.lc.ConsumeLogs(ctx, ld)
}

func (wrapper *exporterWrapper) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return wrapper.tc.ConsumeTraces(ctx, td)
}

func (wrapper *exporterWrapper) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return wrapper.mc.ConsumeMetrics(ctx, md)
}

func (wrapper *exporterWrapper) Capabilities() consumer.Capabilities {
	switch wrapper.inputType {
	case config.LogsDataType:
		return wrapper.lc.Capabilities()
	case config.MetricsDataType:
		return wrapper.mc.Capabilities()
	case config.TracesDataType:
		return wrapper.Capabilities()
	}
	return consumer.Capabilities{}
}

func (wrapper *exporterWrapper) Start(ctx context.Context, host component.Host) error {
	return wrapper.exporter.Start(ctx, host)
}

func (wrapper *exporterWrapper) Shutdown(ctx context.Context) error {
	return wrapper.exporter.Shutdown(ctx)
}
