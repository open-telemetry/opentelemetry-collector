package consumermock

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Nil implements all consumer interfaces but drops all incoming data.
var Nil = &nilConsumer{}

var _ consumer.TraceConsumer = (*nilConsumer)(nil)
var _ consumer.TraceConsumerOld = (*nilConsumer)(nil)
var _ consumer.MetricsConsumer = (*nilConsumer)(nil)
var _ consumer.MetricsConsumerOld = (*nilConsumer)(nil)

type nilConsumer struct {
}

func (n nilConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return nil
}

func (n nilConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return nil
}

func (n nilConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return nil
}

func (n nilConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return nil
}

