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

package metrics

import (
	"context"

	collectormetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterr"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/obsreport"
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
func New(instanceName string, nextConsumer consumer.MetricsConsumer) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, componenterr.ErrNilNextConsumer
	}
	r := &Receiver{
		instanceName: instanceName,
		nextConsumer: nextConsumer,
	}
	return r, nil
}

const (
	receiverTagValue  = "otlp_metrics"
	receiverTransport = "grpc"
)

func (r *Receiver) Export(ctx context.Context, req *collectormetrics.ExportMetricsServiceRequest) (*collectormetrics.ExportMetricsServiceResponse, error) {
	receiverCtx := obsreport.ReceiverContext(ctx, r.instanceName, receiverTransport, receiverTagValue)

	md := data.MetricDataFromOtlp(req.ResourceMetrics)

	err := r.sendToNextConsumer(receiverCtx, md)
	if err != nil {
		return nil, err
	}

	return &collectormetrics.ExportMetricsServiceResponse{}, nil
}

func (r *Receiver) sendToNextConsumer(ctx context.Context, md data.MetricData) error {
	metricCount, dataPointCount := md.MetricAndDataPointCount()
	if metricCount == 0 {
		return nil
	}

	ctx = obsreport.StartMetricsReceiveOp(ctx, r.instanceName, receiverTransport)
	err := r.nextConsumer.ConsumeMetrics(ctx, md)
	obsreport.EndMetricsReceiveOp(ctx, dataFormatProtobuf, dataPointCount, metricCount, err)

	return err
}
