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

// This file contains helpers that are useful to add observability
// with metrics and tracing using OpenCensus to the various pieces
// of the service.

import (
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"google.golang.org/grpc"
)

var (
	mReceiverReceivedSpans      = stats.Int64("otelcol/receiver/received_spans", "Counts the number of spans received by the receiver", "1")
	mReceiverDroppedSpans       = stats.Int64("otelcol/receiver/dropped_spans", "Counts the number of spans dropped by the receiver", "1")
	mReceiverReceivedTimeSeries = stats.Int64("otelcol/receiver/received_timeseries", "Counts the number of timeseries received by the receiver", "1")
	mReceiverDroppedTimeSeries  = stats.Int64("otelcol/receiver/dropped_timeseries", "Counts the number of timeseries dropped by the receiver", "1")

	mExporterReceivedSpans      = stats.Int64("otelcol/exporter/received_spans", "Counts the number of spans received by the exporter", "1")
	mExporterDroppedSpans       = stats.Int64("otelcol/exporter/dropped_spans", "Counts the number of spans dropped by the exporter", "1")
	mExporterReceivedTimeSeries = stats.Int64("otelcol/exporter/received_timeseries", "Counts the number of timeseries received by the exporter", "1")
	mExporterDroppedTimeSeries  = stats.Int64("otelcol/exporter/dropped_timeseries", "Counts the number of timeseries received by the exporter", "1")
	mExporterReceivedLogRecords = stats.Int64("otelcol/exporter/received_logs", "Counts the number of log records received by the exporter", "1")
	mExporterDroppedLogRecords  = stats.Int64("otelcol/exporter/dropped_logs", "Counts the number of log records dropped by the exporter", "1")
)

// LegacyTagKeyReceiver defines tag key for Receiver.
var LegacyTagKeyReceiver, _ = tag.NewKey("otelsvc_receiver")

// LegacyTagKeyExporter defines tag key for Exporter.
var LegacyTagKeyExporter, _ = tag.NewKey("otelsvc_exporter")

// LegacyViewReceiverReceivedSpans defines the view for the receiver received spans metric.
var LegacyViewReceiverReceivedSpans = &view.View{
	Name:        mReceiverReceivedSpans.Name(),
	Description: mReceiverReceivedSpans.Description(),
	Measure:     mReceiverReceivedSpans,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver},
}

// LegacyViewReceiverDroppedSpans defines the view for the receiver dropped spans metric.
var LegacyViewReceiverDroppedSpans = &view.View{
	Name:        mReceiverDroppedSpans.Name(),
	Description: mReceiverDroppedSpans.Description(),
	Measure:     mReceiverDroppedSpans,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver},
}

// LegacyViewReceiverReceivedTimeSeries defines the view for the receiver received timeseries metric.
var LegacyViewReceiverReceivedTimeSeries = &view.View{
	Name:        mReceiverReceivedTimeSeries.Name(),
	Description: mReceiverReceivedTimeSeries.Description(),
	Measure:     mReceiverReceivedTimeSeries,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver},
}

// LegacyViewReceiverDroppedTimeSeries defines the view for the receiver dropped timeseries metric.
var LegacyViewReceiverDroppedTimeSeries = &view.View{
	Name:        mReceiverDroppedTimeSeries.Name(),
	Description: mReceiverDroppedTimeSeries.Description(),
	Measure:     mReceiverDroppedTimeSeries,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver},
}

// LegacyViewExporterReceivedSpans defines the view for the exporter received spans metric.
var LegacyViewExporterReceivedSpans = &view.View{
	Name:        mExporterReceivedSpans.Name(),
	Description: mExporterReceivedSpans.Description(),
	Measure:     mExporterReceivedSpans,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyViewExporterDroppedSpans defines the view for the exporter dropped spans metric.
var LegacyViewExporterDroppedSpans = &view.View{
	Name:        mExporterDroppedSpans.Name(),
	Description: mExporterDroppedSpans.Description(),
	Measure:     mExporterDroppedSpans,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyViewExporterReceivedTimeSeries defines the view for the exporter received timeseries metric.
var LegacyViewExporterReceivedTimeSeries = &view.View{
	Name:        mExporterReceivedTimeSeries.Name(),
	Description: mExporterReceivedTimeSeries.Description(),
	Measure:     mExporterReceivedTimeSeries,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyViewExporterDroppedTimeSeries defines the view for the exporter dropped timeseries metric.
var LegacyViewExporterDroppedTimeSeries = &view.View{
	Name:        mExporterDroppedTimeSeries.Name(),
	Description: mExporterDroppedTimeSeries.Description(),
	Measure:     mExporterDroppedTimeSeries,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyViewExporterReceivedLogRecords defines the view for the exporter received logs metric.
var LegacyViewExporterReceivedLogRecords = &view.View{
	Name:        mExporterReceivedLogRecords.Name(),
	Description: mExporterReceivedLogRecords.Description(),
	Measure:     mExporterReceivedLogRecords,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyViewExporterDroppedLogRecords defines the view for the exporter dropped logs metric.
var LegacyViewExporterDroppedLogRecords = &view.View{
	Name:        mExporterDroppedLogRecords.Name(),
	Description: mExporterDroppedLogRecords.Description(),
	Measure:     mExporterDroppedLogRecords,
	Aggregation: view.Sum(),
	TagKeys:     []tag.Key{LegacyTagKeyReceiver, LegacyTagKeyExporter},
}

// LegacyAllViews has the views for the metrics provided by the agent.
var LegacyAllViews = []*view.View{
	LegacyViewReceiverReceivedSpans,
	LegacyViewReceiverDroppedSpans,
	LegacyViewReceiverReceivedTimeSeries,
	LegacyViewReceiverDroppedTimeSeries,
	LegacyViewExporterReceivedSpans,
	LegacyViewExporterDroppedSpans,
	LegacyViewExporterReceivedLogRecords,
	LegacyViewExporterDroppedLogRecords,
	LegacyViewExporterReceivedTimeSeries,
	LegacyViewExporterDroppedTimeSeries,
}

// GRPCServerWithObservabilityEnabled creates a gRPC server that at a bare minimum has
// the OpenCensus ocgrpc server stats handler enabled for tracing and stats.
// Use it instead of invoking grpc.NewServer directly.
func GRPCServerWithObservabilityEnabled(extraOpts ...grpc.ServerOption) *grpc.Server {
	opts := append(extraOpts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	return grpc.NewServer(opts...)
}
