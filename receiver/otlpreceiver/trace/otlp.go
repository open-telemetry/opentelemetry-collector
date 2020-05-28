// Copyright The OpenTelemetry Authors
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

package trace

import (
	"context"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	collectortrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	dataFormatProtobuf = "protobuf"
)

// Receiver is the type used to handle spans from OpenTelemetry exporters.
type Receiver struct {
	instanceName string
	nextConsumer consumer.TraceConsumer
}

// New creates a new Receiver reference.
func New(instanceName string, nextConsumer consumer.TraceConsumer) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}

	r := &Receiver{
		instanceName: instanceName,
		nextConsumer: nextConsumer,
	}

	return r, nil
}

const (
	receiverTagValue  = "otlp_trace"
	receiverTransport = "grpc"
)

func (r *Receiver) Export(ctx context.Context, req *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	// We need to ensure that it propagates the receiver name as a tag
	ctxWithReceiverName := obsreport.ReceiverContext(ctx, r.instanceName, receiverTransport, receiverTagValue)

	td := pdata.TracesFromOtlp(req.ResourceSpans)
	err := r.sendToNextConsumer(ctxWithReceiverName, td)
	if err != nil {
		return nil, err
	}

	return &collectortrace.ExportTraceServiceResponse{}, nil
}

func (r *Receiver) sendToNextConsumer(ctx context.Context, td pdata.Traces) error {
	if c, ok := client.FromGRPC(ctx); ok {
		ctx = client.NewContext(ctx, c)
	}

	ctx = obsreport.StartTraceDataReceiveOp(
		ctx,
		r.instanceName,
		receiverTransport)

	var consumerErr error
	numSpans := td.SpanCount()
	if numSpans != 0 {
		consumerErr = r.nextConsumer.ConsumeTraces(ctx, td)
	}

	obsreport.EndTraceDataReceiveOp(ctx, dataFormatProtobuf, numSpans, consumerErr)

	return consumerErr
}
