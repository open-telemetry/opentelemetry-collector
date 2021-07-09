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

package obsmetrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

const (
	// PipelineKey is the name of the pipeline
	PipelineKey = "pipeline"
)

var (
	// TagKeyPipeline is the name of the pipeline
	TagKeyPipeline, _ = tag.NewKey(PipelineKey)
	// PipelineProcessingDuration records the an observation for each
	// received batch each time it is exported. For example, a pipeline with
	// one receiver and two exporters will record two observations for each
	// batch of telemetry received: one observation for the time taken to be
	// exported by each expoter.
	// It records an observation for each batch received, not for each batch
	// sent. For example, if five received batches of telemetry are merged
	// together by the "batch" processor and exported together it will still
	// result in five observations: one observation starting from the time each
	// of the five original batches was received.
	PipelineProcessingDuration = stats.Float64(
		"pipeline_processing_duration_seconds",
		"Duration of between when a batch of telemetry in the pipeline was received, and when it was sent by an exporter.",
		stats.UnitSeconds)
)
