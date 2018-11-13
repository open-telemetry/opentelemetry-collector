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

package metricsink

import (
	"context"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricpb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
)

// Sink is an interface that receives metrics from a Node identifier.
type Sink interface {
	ReceiveMetrics(ctx context.Context, node *commonpb.Node, resource *resourcepb.Resource, metrics ...*metricpb.Metric) (*Acknowledgement, error)
}

// Acknowledgement struct reports the number of saved and dropped spans in a
// ReceiveSpans call.
type Acknowledgement struct {
	SavedMetrics   uint64
	DroppedMetrics uint64
}
