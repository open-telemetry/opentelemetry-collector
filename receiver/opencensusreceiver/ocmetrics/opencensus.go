// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ocmetrics

import (
	"context"
	"errors"
	"io"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"google.golang.org/api/support/bundler"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// Receiver is the type used to handle metrics from OpenCensus exporters.
type Receiver struct {
	instanceName       string
	nextConsumer       consumer.MetricsConsumer
	metricBufferPeriod time.Duration
	metricBufferCount  int
}

// New creates a new ocmetrics.Receiver reference.
func New(instanceName string, nextConsumer consumer.MetricsConsumer, opts ...Option) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}
	ocr := &Receiver{
		instanceName: instanceName,
		nextConsumer: nextConsumer,
	}
	for _, opt := range opts {
		opt.WithReceiver(ocr)
	}
	return ocr, nil
}

var _ agentmetricspb.MetricsServiceServer = (*Receiver)(nil)

var errMetricsExportProtocolViolation = errors.New("protocol violation: Export's first message must have a Node")

const (
	receiverTagValue  = "oc_metrics"
	receiverTransport = "grpc" // TODO: transport is being hard coded for now, investigate if info is available on context.
)

// Export is the gRPC method that receives streamed metrics from
// OpenCensus-metricproto compatible libraries/applications.
func (ocr *Receiver) Export(mes agentmetricspb.MetricsService_ExportServer) error {
	// The bundler will receive batches of metrics i.e. []*metricspb.Metric
	ctxWithReceiverName := obsreport.ReceiverContext(
		mes.Context(), ocr.instanceName, receiverTransport, receiverTagValue)
	metricsBundler := bundler.NewBundler((*consumerdata.MetricsData)(nil), func(payload interface{}) {
		ocr.batchMetricExporting(ctxWithReceiverName, payload)
	})

	metricBufferPeriod := ocr.metricBufferPeriod
	if metricBufferPeriod <= 0 {
		metricBufferPeriod = 2 * time.Second // Arbitrary value
	}
	metricBufferCount := ocr.metricBufferCount
	if metricBufferCount <= 0 {
		// TODO: (@odeke-em) provide an option to disable any buffering
		metricBufferCount = 50 // Arbitrary value
	}

	metricsBundler.DelayThreshold = metricBufferPeriod
	metricsBundler.BundleCountThreshold = metricBufferCount

	// Retrieve the first message. It MUST have a non-nil Node.
	recv, err := mes.Recv()
	if err != nil {
		return err
	}

	// Check the condition that the first message has a non-nil Node.
	if recv.Node == nil {
		return errMetricsExportProtocolViolation
	}

	var lastNonNilNode *commonpb.Node
	var resource *resourcepb.Resource
	// Now that we've got the first message with a Node, we can start to receive streamed up metrics.
	for {
		// If a Node has been sent from downstream, save and use it.
		if recv.Node != nil {
			lastNonNilNode = recv.Node
		}

		// TODO(songya): differentiate between unset and nil resource. See
		// https://github.com/census-instrumentation/opencensus-proto/issues/146.
		if recv.Resource != nil {
			resource = recv.Resource
		}

		processReceivedMetrics(lastNonNilNode, resource, recv.Metrics, metricsBundler)

		recv, err = mes.Recv()
		if err != nil {
			if err == io.EOF {
				// Do not return EOF as an error so that grpc-gateway calls get an empty
				// response with HTTP status code 200 rather than a 500 error with EOF.
				return nil
			}
			return err
		}
	}
}

func processReceivedMetrics(ni *commonpb.Node, resource *resourcepb.Resource, metrics []*metricspb.Metric, bundler *bundler.Bundler) {
	// Firstly, we'll add them to the bundler.
	if len(metrics) > 0 {
		bundlerPayload := &consumerdata.MetricsData{Node: ni, Metrics: metrics, Resource: resource}
		bundler.Add(bundlerPayload, len(bundlerPayload.Metrics))
	}
}

func (ocr *Receiver) batchMetricExporting(longLivedRPCCtx context.Context, payload interface{}) {
	ctx, span := obsreport.StartMetricsReceiveOp(
		context.Background(),
		ocr.instanceName,
		receiverTransport)

	// If the starting RPC has a parent span, then add it as a parent link.
	obsreport.SetParentLink(longLivedRPCCtx, span)

	mds := payload.([]*consumerdata.MetricsData)
	numTimeSeries := 0
	numPoints := 0
	var consumerErr error
	if len(mds) != 0 {
		for _, md := range mds {
			if md == nil || len(md.Metrics) == 0 {
				continue
			}

			// Legacy count number of time series, there was no legacy metric
			// here before but count it anyway for consistency
			for _, metric := range md.Metrics {
				numTimeSeries += len(metric.Timeseries)
				for _, ts := range metric.GetTimeseries() {
					numPoints += len(ts.GetPoints())
				}
			}

			if consumerErr != nil {
				continue
			}
			consumerErr = ocr.nextConsumer.ConsumeMetricsData(ctx, *md)
		}
	}

	obsreport.EndMetricsReceiveOp(
		ctx,
		span,
		"protobuf",
		numPoints,
		numTimeSeries,
		consumerErr)
}
