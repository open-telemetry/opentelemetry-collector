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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// startReceiveOptions has the options related to starting a receive operation.
type startReceiveOptions struct {
	// LongLivedCtx when true indicates that the context passed in the call
	// outlives the individual receive operation. See WithLongLivedCtx() for
	// more information.
	LongLivedCtx bool
}

// StartReceiveOption function applues changes to startReceiveOptions.
type StartReceiveOption func(*startReceiveOptions)

// WithLongLivedCtx indicates that the context passed in the call outlives the
// receive operation at hand. Typically the long lived context is associated
// to a connection, eg.: a gRPC stream or a TCP connection, for which many
// batches of data are received in individual operations without a corresponding
// new context per operation.
//
// Example:
//
//    func (r *receiver) ClientConnect(ctx context.Context, rcvChan <-chan pdata.Traces) {
//        longLivedCtx := obsreport.ReceiverContext(ctx, r.config.Name(), r.transport, "")
//        for {
//            // Since the context outlives the individual receive operations call obsreport using
//            // WithLongLivedCtx().
//            ctx := obsreport.StartTracesOp(
//                longLivedCtx,
//                r.config.Name(),
//                r.transport,
//                obsreport.WithLongLivedCtx())
//
//            td, ok := <-rcvChan
//            var err error
//            if ok {
//                err = r.nextConsumer.ConsumeTraces(ctx, td)
//            }
//            obsreport.EndTracesOp(
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
	return func(opts *startReceiveOptions) {
		opts.LongLivedCtx = true
	}
}

// Receiver is a helper to add obersvability to a component.Receiver.
type Receiver struct {
	receiverID config.ComponentID
	transport  string
}

// ReceiverSettings are settings for creating an Receiver.
type ReceiverSettings struct {
	ReceiverID config.ComponentID
	Transport  string
}

// NewReceiver creates a new Receiver.
func NewReceiver(cfg ReceiverSettings) *Receiver {
	return &Receiver{
		receiverID: cfg.ReceiverID,
		transport:  cfg.Transport,
	}
}

// StartTracesOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartTracesOp(operationCtx context.Context, opt ...StartReceiveOption) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiveTraceDataOperationSuffix, opt...)
}

// EndTracesOp completes the receive operation that was started with
// StartTracesOp.
func (rec *Receiver) EndTracesOp(
	receiverCtx context.Context,
	format string,
	numReceivedSpans int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedSpans, err, config.TracesDataType)
}

// StartLogsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartLogsOp(operationCtx context.Context, opt ...StartReceiveOption) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverLogsOperationSuffix, opt...)
}

// EndLogsOp completes the receive operation that was started with
// StartLogsOp.
func (rec *Receiver) EndLogsOp(
	receiverCtx context.Context,
	format string,
	numReceivedLogRecords int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedLogRecords, err, config.LogsDataType)
}

// StartMetricsOp is called when a request is received from a client.
// The returned context should be used in other calls to the obsreport functions
// dealing with the same receive operation.
func (rec *Receiver) StartMetricsOp(operationCtx context.Context, opt ...StartReceiveOption) context.Context {
	return rec.startOp(operationCtx, obsmetrics.ReceiverMetricsOperationSuffix, opt...)
}

// EndMetricsOp completes the receive operation that was started with
// StartMetricsOp.
func (rec *Receiver) EndMetricsOp(
	receiverCtx context.Context,
	format string,
	numReceivedPoints int,
	err error,
) {
	rec.endOp(receiverCtx, format, numReceivedPoints, err, config.MetricsDataType)
}

// ReceiverContext adds the keys used when recording observability metrics to
// the given context returning the newly created context. This context should
// be used in related calls to the obsreport functions so metrics are properly
// recorded.
func ReceiverContext(
	ctx context.Context,
	receiverID config.ComponentID,
	transport string,
) context.Context {
	ctx, _ = tag.New(ctx,
		tag.Upsert(obsmetrics.TagKeyReceiver, receiverID.String(), tag.WithTTL(tag.TTLNoPropagation)),
		tag.Upsert(obsmetrics.TagKeyTransport, transport, tag.WithTTL(tag.TTLNoPropagation)))

	return ctx
}

// startOp creates the span used to trace the operation. Returning
// the updated context with the created span.
func (rec *Receiver) startOp(
	receiverCtx context.Context,
	operationSuffix string,
	opt ...StartReceiveOption,
) context.Context {
	var opts startReceiveOptions
	for _, o := range opt {
		o(&opts)
	}

	var ctx context.Context
	var span *trace.Span
	spanName := obsmetrics.ReceiverPrefix + rec.receiverID.String() + operationSuffix
	if !opts.LongLivedCtx {
		ctx, span = trace.StartSpan(receiverCtx, spanName)
	} else {
		// Since the receiverCtx is long lived do not use it to start the span.
		// This way this trace ends when the EndTracesOp is called.
		// Here is safe to ignore the returned context since it is not used below.
		_, span = trace.StartSpan(context.Background(), spanName)

		// If the long lived context has a parent span, then add it as a parent link.
		setParentLink(receiverCtx, span)

		ctx = trace.NewContext(receiverCtx, span)
	}

	if rec.transport != "" {
		span.AddAttributes(trace.StringAttribute(obsmetrics.TransportKey, rec.transport))
	}
	return ctx
}

// endOp records the observability signals at the end of an operation.
func (rec *Receiver) endOp(
	receiverCtx context.Context,
	format string,
	numReceivedItems int,
	err error,
	dataType config.DataType,
) {
	numAccepted := numReceivedItems
	numRefused := 0
	if err != nil {
		numAccepted = 0
		numRefused = numReceivedItems
	}

	span := trace.FromContext(receiverCtx)

	if obsreportconfig.Level != configtelemetry.LevelNone {
		var acceptedMeasure, refusedMeasure *stats.Int64Measure
		switch dataType {
		case config.TracesDataType:
			acceptedMeasure = obsmetrics.ReceiverAcceptedSpans
			refusedMeasure = obsmetrics.ReceiverRefusedSpans
		case config.MetricsDataType:
			acceptedMeasure = obsmetrics.ReceiverAcceptedMetricPoints
			refusedMeasure = obsmetrics.ReceiverRefusedMetricPoints
		case config.LogsDataType:
			acceptedMeasure = obsmetrics.ReceiverAcceptedLogRecords
			refusedMeasure = obsmetrics.ReceiverRefusedLogRecords
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
		case config.TracesDataType:
			acceptedItemsKey = obsmetrics.AcceptedSpansKey
			refusedItemsKey = obsmetrics.RefusedSpansKey
		case config.MetricsDataType:
			acceptedItemsKey = obsmetrics.AcceptedMetricPointsKey
			refusedItemsKey = obsmetrics.RefusedMetricPointsKey
		case config.LogsDataType:
			acceptedItemsKey = obsmetrics.AcceptedLogRecordsKey
			refusedItemsKey = obsmetrics.RefusedLogRecordsKey
		}

		span.AddAttributes(
			trace.StringAttribute(
				obsmetrics.FormatKey, format),
			trace.Int64Attribute(
				acceptedItemsKey, int64(numAccepted)),
			trace.Int64Attribute(
				refusedItemsKey, int64(numRefused)),
		)
		span.SetStatus(errToStatus(err))
	}
	span.End()
}
