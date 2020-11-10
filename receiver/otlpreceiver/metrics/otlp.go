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

package metrics

import (
	"context"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	collectormetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	dataFormatProtobuf = "protobuf"
)

// Receiver is the type used to handle metrics from OpenTelemetry exporters.
type Receiver struct {
	instanceName string
	nextConsumer consumer.MetricsConsumer
}

// New creates a new Receiver reference.
func New(instanceName string, nextConsumer consumer.MetricsConsumer) *Receiver {
	r := &Receiver{
		instanceName: instanceName,
		nextConsumer: nextConsumer,
	}
	return r
}

const (
	receiverTagValue  = "otlp_metrics"
	receiverTransport = "grpc"
)

func (r *Receiver) Export(ctx context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	receiverCtx := obsreport.ReceiverContext(ctx, r.instanceName, receiverTransport)

	md := pdata.MetricsFromOtlp(req.ResourceMetrics)

	err := r.sendToNextConsumer(receiverCtx, md)
	if err != nil {
		return nil, err
	}

	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func (r *Receiver) sendToNextConsumer(ctx context.Context, md pdata.Metrics) error {
	metricCount, dataPointCount := md.MetricAndDataPointCount()
	if metricCount == 0 {
		return nil
	}

	if c, ok := client.FromGRPC(ctx); ok {
		ctx = client.NewContext(ctx, c)
	}

	ctx = obsreport.StartMetricsReceiveOp(ctx, r.instanceName, receiverTransport)
	err := r.nextConsumer.ConsumeMetrics(ctx, md)
	obsreport.EndMetricsReceiveOp(ctx, dataFormatProtobuf, dataPointCount, err)

	return err
}
