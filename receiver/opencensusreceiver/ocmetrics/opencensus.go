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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// Receiver is the type used to handle metrics from OpenCensus exporters.
type Receiver struct {
	instanceName string
	nextConsumer consumer.MetricsConsumer
}

// New creates a new ocmetrics.Receiver reference.
func New(instanceName string, nextConsumer consumer.MetricsConsumer) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}
	ocr := &Receiver{
		instanceName: instanceName,
		nextConsumer: nextConsumer,
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
	receiverCtx := obsreport.ReceiverContext(
		mes.Context(), ocr.instanceName, receiverTransport, receiverTagValue)

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

		ocr.processReceivedMetrics(receiverCtx, lastNonNilNode, resource, recv.Metrics)

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

func (ocr *Receiver) processReceivedMetrics(longLivedRPCCtx context.Context, ni *commonpb.Node, resource *resourcepb.Resource, metrics []*metricspb.Metric) {
	if len(metrics) > 0 {
		md := consumerdata.MetricsData{Node: ni, Metrics: metrics, Resource: resource}
		ocr.sendToNextConsumer(longLivedRPCCtx, md)
	}
}

func (ocr *Receiver) sendToNextConsumer(longLivedRPCCtx context.Context, md consumerdata.MetricsData) {
	// Do not use longLivedRPCCtx to start the span so this trace ends right at this
	// function, and the span is not a child of any span from the stream context.
	_, span := obsreport.StartMetricsReceiveOp(
		context.Background(),
		ocr.instanceName,
		receiverTransport)

	// If the starting RPC has a parent span, then add it as a parent link.
	obsreport.SetParentLink(longLivedRPCCtx, span)

	numTimeSeries := 0
	numPoints := 0
	// Count number of time series and data points.
	for _, metric := range md.Metrics {
		numTimeSeries += len(metric.Timeseries)
		for _, ts := range metric.GetTimeseries() {
			numPoints += len(ts.GetPoints())
		}
	}

	consumerErr := ocr.nextConsumer.ConsumeMetricsData(longLivedRPCCtx, md)

	obsreport.EndMetricsReceiveOp(
		longLivedRPCCtx,
		span,
		"protobuf",
		numPoints,
		numTimeSeries,
		consumerErr)
}
