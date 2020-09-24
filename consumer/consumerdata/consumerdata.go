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

// Package consumerdata contains data structures that holds proto metrics/spans, node and resource.
package consumerdata

import (
	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

// MetricsData is a struct that groups proto metrics with a unique node and a resource.
// Deprecated: use pdata.Metrics instead.
type MetricsData struct {
	Node     *commonpb.Node
	Resource *resourcepb.Resource
	Metrics  []*metricspb.Metric
}

// TraceData is a struct that groups proto spans with a unique node and a resource.
// Deprecated: use pdata.Traces instead.
type TraceData struct {
	Node         *commonpb.Node
	Resource     *resourcepb.Resource
	Spans        []*tracepb.Span
	SourceFormat string
}
