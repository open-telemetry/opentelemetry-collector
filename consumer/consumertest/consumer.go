package consumertest

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Consumer is a convenience interface that implements all consumer interfaces.
// It has a private function on it to forbid external users to implement it,
// to allow us to add extra functions without breaking compatibility because
// nobody else implements this interface.
type Consumer interface {
	// ConsumeTraces to implement the consumer.Traces.
	ConsumeTraces(context.Context, pdata.Traces) error
	// ConsumeMetrics to implement the consumer.Metrics.
	ConsumeMetrics(context.Context, pdata.Metrics) error
	// ConsumeLogs to implement the consumer.Logs.
	ConsumeLogs(context.Context, pdata.Logs) error
	unexported()
}

var _ consumer.Logs = (Consumer)(nil)
var _ consumer.Metrics = (Consumer)(nil)
var _ consumer.Traces = (Consumer)(nil)
