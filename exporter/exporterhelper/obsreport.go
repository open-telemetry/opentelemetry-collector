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

package exporterhelper

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
	"go.opentelemetry.io/collector/obsreport"
)

// TODO: Incorporate this functionality along with tests from obsreport_test.go
//       into existing `obsreport` package once its functionally is not exposed
//       as public API. For now this part is kept private.

// obsExporter is a helper to add observability to a component.Exporter.
type obsExporter struct {
	*obsreport.Exporter
	mutators []tag.Mutator
}

// newObsExporter creates a new observability exporter.
func newObsExporter(cfg obsreport.ExporterSettings) *obsExporter {
	return &obsExporter{
		obsreport.NewExporter(cfg),
		[]tag.Mutator{tag.Upsert(obsmetrics.TagKeyExporter, cfg.ExporterID.String(), tag.WithTTL(tag.TTLNoPropagation))},
	}
}

// recordTracesEnqueueFailure records number of spans that failed to be added to the sending queue.
func (eor *obsExporter) recordTracesEnqueueFailure(ctx context.Context, numSpans int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueSpans.M(int64(numSpans)))
}

// recordMetricsEnqueueFailure records number of metric points that failed to be added to the sending queue.
func (eor *obsExporter) recordMetricsEnqueueFailure(ctx context.Context, numMetricPoints int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueMetricPoints.M(int64(numMetricPoints)))
}

// recordLogsEnqueueFailure records number of log records that failed to be added to the sending queue.
func (eor *obsExporter) recordLogsEnqueueFailure(ctx context.Context, numLogRecords int) {
	_ = stats.RecordWithTags(ctx, eor.mutators, obsmetrics.ExporterFailedToEnqueueLogRecords.M(int64(numLogRecords)))
}
