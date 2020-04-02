// Copyright 2020, OpenTelemetry Authors
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

	collectortrace "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/client"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
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
		return nil, oterr.ErrNilNextConsumer
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

	td := data.TraceDataFromOtlp(req.ResourceSpans)
	err := r.sendToNextConsumer(ctxWithReceiverName, td)
	if err != nil {
		return nil, err
	}

	return &collectortrace.ExportTraceServiceResponse{}, nil
}

func (r *Receiver) sendToNextConsumer(ctx context.Context, td data.TraceData) error {
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
		consumerErr = r.nextConsumer.ConsumeTrace(ctx, td)
	}

	obsreport.EndTraceDataReceiveOp(ctx, dataFormatProtobuf, numSpans, consumerErr)

	return consumerErr
}
