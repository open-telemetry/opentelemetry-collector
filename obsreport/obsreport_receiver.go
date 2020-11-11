// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

const (
	// Key used to identify receivers in metrics and traces.
	ReceiverKey = "receiver"
	// Key used to identify the transport used to received the data.
	TransportKey = "transport"
	// Key used to identify the format of the data received.
	FormatKey = "format"

	// Key used to identify spans accepted by the Collector.
	AcceptedSpansKey = "accepted_spans"
	// Key used to identify spans refused (ie.: not ingested) by the Collector.
	RefusedSpansKey = "refused_spans"

	// Key used to identify metric points accepted by the Collector.
	AcceptedMetricPointsKey = "accepted_metric_points"
	// Key used to identify metric points refused (ie.: not ingested) by the
	// Collector.
	RefusedMetricPointsKey = "refused_metric_points"

	// Key used to identify log records accepted by the Collector.
	AcceptedLogRecordsKey = "accepted_log_records"
	// Key used to identify log records refused (ie.: not ingested) by the
	// Collector.
	RefusedLogRecordsKey = "refused_log_records"
)

var (
	tagKeyReceiver, _  = tag.NewKey(ReceiverKey)
	tagKeyTransport, _ = tag.NewKey(TransportKey)

	receiverPrefix                  = ReceiverKey + nameSep
	receiveTraceDataOperationSuffix = nameSep + "TraceDataReceived"
	receiverMetricsOperationSuffix  = nameSep + "MetricsReceived"
	receiverLogsOperationSuffix     = nameSep + "LogsReceived"

	// Receiver metrics. Any count of data items below is in the original format
	// that they were received, reasoning: reconciliation is easier if measurements
	// on clients and receiver are expected to be the same. Translation issues
	// that result in a different number of elements should be reported in a
	// separate way.
	mReceiverAcceptedSpans = stats.Int64(
		receiverPrefix+AcceptedSpansKey,
		"Number of spans successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverRefusedSpans = stats.Int64(
		receiverPrefix+RefusedSpansKey,
		"Number of spans that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverAcceptedMetricPoints = stats.Int64(
		receiverPrefix+AcceptedMetricPointsKey,
		"Number of metric points successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverRefusedMetricPoints = stats.Int64(
		receiverPrefix+RefusedMetricPointsKey,
		"Number of metric points that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverAcceptedLogRecords = stats.Int64(
		receiverPrefix+AcceptedLogRecordsKey,
		"Number of log records successfully pushed into the pipeline.",
		stats.UnitDimensionless)
	mReceiverRefusedLogRecords = stats.Int64(
		receiverPrefix+RefusedLogRecordsKey,
		"Number of log records that could not be pushed into the pipeline.",
		stats.UnitDimensionless)
)

// StartReceiveOptions has the options related to starting a receive operation.
type StartReceiveOptions struct {
	// LongLivedCtx when true indicates that the context passed in the call
	// outlives the individual receive operation. See WithLongLivedCtx() for
	// more information.
	LongLivedCtx bool
}

// StartReceiveOption function applues changes to StartReceiveOptions.
type StartReceiveOption func(*StartReceiveOptions)

// WithLongLivedCtx indicates that the context passed in the call outlives the
// receive operation at hand. Typically the long lived context is associated
// to a connection, eg.: a gRPC stream or a TCP connection, for which many
// batches of data are received in individual operations without a corresponding
// new context per operation.
//
// Example:
//
//    func (r *receiver) ClientConnect(ctx context.Context, rcvChan <-chan consumerdata.TraceData) {
//        longLivedCtx := obsreport.ReceiverContext(ctx, r.config.Name(), r.transport, "")
//        for {
//            // Since the context outlives the individual receive operations call obsreport using
//            // WithLongLivedCtx().
//            ctx := obsreport.StartTraceDataReceiveOp(
//                longLivedCtx,
//                r.config.Name(),
//                r.transport,
//                obsreport.WithLongLivedCtx())
//
//            td, ok := <-rcvChan
//            var err error
//            if ok {
//                err = r.nextConsumer.ConsumeTraceData(ctx, td)
//            }
//            obsreport.EndTraceDataReceiveOp(
//                ctx,
//                r.format,
//                len(td.Spans),
//                err)
//            if !ok {
//                break
//            }
//        }
//    }
//
func WithLongLivedCtx() StartReceiveOption {
	return func(opts *StartReceiveOptions) {
		opts.LongLivedCtx = true
	}
}

// StartTraceDataReceiveOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func StartTraceDataReceiveOp(
	operationCtx context.Context,
	receiver string,
	transport string,
	opt ...StartReceiveOption,
) context.Context {
	return traceReceiveOp(
		operationCtx,
		receiver,
		transport,
		receiveTraceDataOperationSuffix,
		opt...)
}

// EndTraceDataReceiveOp completes the receive operation that was started with
// StartTraceDataReceiveOp.
func EndTraceDataReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	endReceiveOp(
		receiverCtx,
		format,
		numReceivedSpans,
		err,
		configmodels.TracesDataType,
	)
}

// StartLogsReceiveOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func StartLogsReceiveOp(
	operationCtx context.Context,
	receiver string,
	transport string,
	opt ...StartReceiveOption,
) context.Context {
	return traceReceiveOp(
		operationCtx,
		receiver,
		transport,
		receiverLogsOperationSuffix,
		opt...)
}

// EndLogsReceiveOp completes the receive operation that was started with
// StartLogsReceiveOp.
func EndLogsReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedLogRecords int,
	err error,
) {
	endReceiveOp(
		receiverCtx,
		format,
		numReceivedLogRecords,
		err,
		configmodels.LogsDataType,
	)
}

// StartMetricsReceiveOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func StartMetricsReceiveOp(
	operationCtx context.Context,
	receiver string,
	transport string,
	opt ...StartReceiveOption,
) context.Context {
	return traceReceiveOp(
		operationCtx,
		receiver,
		transport,
		receiverMetricsOperationSuffix,
		opt...)
}

// EndMetricsReceiveOp completes the receive operation that was started with
// StartMetricsReceiveOp.
func EndMetricsReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	err error,
) {
	endReceiveOp(
		receiverCtx,
		format,
		numReceivedPoints,
		err,
		configmodels.MetricsDataType,
	)
}

// ReceiverContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ReceiverContext(
	ctx context.Context,
	receiver string,
	transport string,
) context.Context {
	ctx, _ = tag.New(ctx,
		tag.Upsert(tagKeyReceiver, receiver, tag.WithTTL(tag.TTLNoPropagation)),
		tag.Upsert(tagKeyTransport, transport, tag.WithTTL(tag.TTLNoPropagation)))

	return ctx
}

// traceReceiveOp creates the span used to trace the operation. Returning
// the updated context with the created span.
func traceReceiveOp(
	receiverCtx context.Context,
	receiverName string,
	transport string,
	operationSuffix string,
	opt ...StartReceiveOption,
) context.Context {
	var opts StartReceiveOptions
	for _, o := range opt {
		o(&opts)
	}

	var ctx context.Context
	var span *trace.Span
	spanName := receiverPrefix + receiverName + operationSuffix
	if !opts.LongLivedCtx {
		ctx, span = trace.StartSpan(receiverCtx, spanName)
	} else {
		// Since the receiverCtx is long lived do not use it to start the span.
		// This way this trace ends when the EndTraceDataReceiveOp is called.
		// Here is safe to ignore the returned context since it is not used below.
		_, span = trace.StartSpan(context.Background(), spanName)

		// If the long lived context has a parent span, then add it as a parent link.
		setParentLink(receiverCtx, span)

		ctx = trace.NewContext(receiverCtx, span)
	}

	if transport != "" {
		span.AddAttributes(trace.StringAttribute(TransportKey, transport))
	}
	return ctx
}

// endReceiveOp records the observability signals at the end of an operation.
func endReceiveOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	dataType configmodels.DataType,
) {
	numAccepted := numReceivedItems
	numRefused := 0
	if err != nil {
		numAccepted = 0
		numRefused = numReceivedItems
	}

	span := trace.FromContext(receiverCtx)

	if gLevel != configtelemetry.LevelNone {
		var acceptedMeasure, refusedMeasure *stats.Int64Measure
		switch dataType {
		case configmodels.TracesDataType:
			acceptedMeasure = mReceiverAcceptedSpans
			refusedMeasure = mReceiverRefusedSpans
		case configmodels.MetricsDataType:
			acceptedMeasure = mReceiverAcceptedMetricPoints
			refusedMeasure = mReceiverRefusedMetricPoints
		case configmodels.LogsDataType:
			acceptedMeasure = mReceiverAcceptedLogRecords
			refusedMeasure = mReceiverRefusedLogRecords
		}

		stats.Record(
			receiverCtx,
			acceptedMeasure.M(int64(numAccepted)),
			refusedMeasure.M(int64(numRefused)))
	}

	// end span according to errors
	if span.IsRecordingEvents() {
		var acceptedItemsKey, refusedItemsKey string
		switch dataType {
		case configmodels.TracesDataType:
			acceptedItemsKey = AcceptedSpansKey
			refusedItemsKey = RefusedSpansKey
		case configmodels.MetricsDataType:
			acceptedItemsKey = AcceptedMetricPointsKey
			refusedItemsKey = RefusedMetricPointsKey
		case configmodels.LogsDataType:
			acceptedItemsKey = AcceptedLogRecordsKey
			refusedItemsKey = RefusedLogRecordsKey
		}

		span.AddAttributes(
			trace.StringAttribute(
				FormatKey, format),
			trace.Int64Attribute(
				acceptedItemsKey, int64(numAccepted)),
			trace.Int64Attribute(
				refusedItemsKey, int64(numRefused)),
		)
		span.SetStatus(errToStatus(err))
	}
	span.End()
}
