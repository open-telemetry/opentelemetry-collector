// Copyright 2018, OpenCensus Authors
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

package exporter

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

// MetricsExporter is an interface that receives data.MetricsData, converts it as needed, and
// sends it to different destinations.
//
// ExportMetricsData receives data.MetricsData for processing by the exporter.
type MetricsExporter interface {
	ExportMetricsData(ctx context.Context, md data.MetricsData) error
}

// MultiMetricsExporters wraps multiple metrics exporters in a single one.
func MultiMetricsExporters(mes ...MetricsExporter) receiver.MetricsReceiverSink {
	return metricsExporters(mes)
}

type metricsExporters []MetricsExporter

var _ receiver.MetricsReceiverSink = (*metricsExporters)(nil)

// ExportMetricsData exports the MetricsData to all exporters wrapped by the current one.
func (mes metricsExporters) ExportMetricsData(ctx context.Context, md data.MetricsData) error {
	for _, me := range mes {
		_ = me.ExportMetricsData(ctx, md)
	}
	return nil
}

// ReceiveTraceData receives the span data in the protobuf format, translates it, and forwards the transformed
// span data to all trace exporters wrapped by the current one.
func (mes metricsExporters) ReceiveMetricsData(ctx context.Context, md data.MetricsData) (*receiver.MetricsReceiverAcknowledgement, error) {
	for _, me := range mes {
		_ = me.ExportMetricsData(ctx, md)
	}

	ack := &receiver.MetricsReceiverAcknowledgement{
		SavedMetrics: uint64(len(md.Metrics)),
	}
	return ack, nil
}
