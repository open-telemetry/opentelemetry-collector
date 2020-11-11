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

package ocmetrics

import (
	"context"
	"errors"
	"io"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// Receiver is the type used to handle metrics from OpenCensus exporters.
type Receiver struct {
	agentmetricspb.UnimplementedMetricsServiceServer
	instanceName string
	nextConsumer consumer.MetricsConsumer
}

// New creates a new ocmetrics.Receiver reference.
func New(instanceName string, nextConsumer consumer.MetricsConsumer) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
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
	receiverTagValue   = "oc_metrics"
	receiverTransport  = "grpc" // TODO: transport is being hard coded for now, investigate if info is available on context.
	receiverDataFormat = "protobuf"
)

// Export is the gRPC method that receives streamed metrics from
// OpenCensus-metricproto compatible libraries/applications.
func (ocr *Receiver) Export(mes agentmetricspb.MetricsService_ExportServer) error {
	longLivedRPCCtx := obsreport.ReceiverContext(mes.Context(), ocr.instanceName, receiverTransport)

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
		lastNonNilNode, resource, err = ocr.processReceivedMsg(
			longLivedRPCCtx,
			lastNonNilNode,
			resource,
			recv)
		if err != nil {
			return err
		}

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

func (ocr *Receiver) processReceivedMsg(
	longLivedRPCCtx context.Context,
	lastNonNilNode *commonpb.Node,
	resource *resourcepb.Resource,
	recv *agentmetricspb.ExportMetricsServiceRequest,
) (*commonpb.Node, *resourcepb.Resource, error) {
	// If a Node has been sent from downstream, save and use it.
	if recv.Node != nil {
		lastNonNilNode = recv.Node
	}

	// TODO(songya): differentiate between unset and nil resource. See
	// https://github.com/census-instrumentation/opencensus-proto/issues/146.
	if recv.Resource != nil {
		resource = recv.Resource
	}

	md := consumerdata.MetricsData{
		Node:     lastNonNilNode,
		Resource: resource,
		Metrics:  recv.Metrics,
	}

	err := ocr.sendToNextConsumer(longLivedRPCCtx, md)
	return lastNonNilNode, resource, err
}

func (ocr *Receiver) sendToNextConsumer(longLivedRPCCtx context.Context, md consumerdata.MetricsData) error {
	ctx := obsreport.StartMetricsReceiveOp(
		longLivedRPCCtx,
		ocr.instanceName,
		receiverTransport,
		obsreport.WithLongLivedCtx())

	numTimeSeries := 0
	numPoints := 0
	// Count number of time series and data points.
	for _, metric := range md.Metrics {
		numTimeSeries += len(metric.Timeseries)
		for _, ts := range metric.GetTimeseries() {
			numPoints += len(ts.GetPoints())
		}
	}

	var consumerErr error
	if len(md.Metrics) > 0 {
		consumerErr = ocr.nextConsumer.ConsumeMetrics(ctx, internaldata.OCToMetrics(md))
	}

	obsreport.EndMetricsReceiveOp(
		ctx,
		receiverDataFormat,
		numPoints,
		consumerErr)

	return consumerErr
}
