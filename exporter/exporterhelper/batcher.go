package exporterhelper

import (
	"context"
	"errors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// NewTracesBatchExporter creates a Traces exporter that batches requests. This is an experimental feature and API.
func NewTracesBatchExporter(
	_ context.Context,
	set exporter.CreateSettings,
	cfg exporterbatcher.Config,
	tbf exporterbatcher.TracesBatchFactory,
	options ...Option,
) (exporter.Traces, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if tbf.BatchFromTraces == nil {
		panic("nil BatchFromTraces")
	}

	bs := fromOptions(options...)
	if bs.QueueSettings.Enabled && bs.QueueSettings.StorageID != nil && tbf.BatchFromBytes == nil {
		return nil, errors.New("the exporter doesn't persistent queue")
	}
	var batchUnmarshaler internal.RequestUnmarshaler
	if tbf.BatchFromBytes != nil {
		batchUnmarshaler = func(bytes []byte) (internal.Request, error) {
			b, err := tbf.BatchFromBytes(bytes)
			return b.(internal.Request), err
		}
	}
	be, err := newBaseExporter(set, bs, component.DataTypeTraces, batchUnmarshaler)
	if err != nil {
		return nil, err
	}
	be.wrapConsumerSender(func(nextSender internal.RequestSender) internal.RequestSender {
		return &tracesExporterWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	bc, err := exporterbatcher.NewTracesConsumer(set, cfg, tbf, be.sender,
		obsreportconfig.UseOtelForInternalMetricsfeatureGate.IsEnabled())
	if err != nil {
		return nil, err
	}

	tc, err := consumer.NewTraces(func(ctx context.Context, td ptrace.Traces) error {
		if be.qrSender.queue.Size() >= bs.QueueSettings.QueueSize {
			be.obsrep.recordTracesEnqueueFailure(ctx, int64(td.SpanCount()))
			return errSendingQueueIsFull
		}
		return bc.ConsumeTraces(ctx, td)
	}, bs.consumerOptions...)

	return &traceExporter{
		baseExporter: be,
		Traces:       tc,
	}, err
}
