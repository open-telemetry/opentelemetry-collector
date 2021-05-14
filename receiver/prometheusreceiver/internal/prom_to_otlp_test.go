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

package internal

import (
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/translator/internaldata"
)

// Parity test to ensure that createNodeAndResource produces identical results to createNodeAndResourcePdata.
func TestCreateNodeAndResourceEquivalence(t *testing.T) {
	job, instance, scheme := "converter", "ocmetrics", "http"
	ocNode, ocResource := createNodeAndResource(job, instance, scheme)
	mdFromOC := internaldata.OCToMetrics(internaldata.MetricsData{
		Node:     ocNode,
		Resource: ocResource,
		// We need to pass in a dummy set of metrics
		// just to populate and allow for full conversion.
		Metrics: []*metricspb.Metric{
			{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "m1",
					Description: "d1",
					Unit:        "By",
				},
			},
		},
	})

	fromOCResource := mdFromOC.ResourceMetrics().At(0).Resource().Attributes().Sort()
	byDirectOTLPResource := createNodeAndResourcePdata(job, instance, scheme).Attributes().Sort()

	require.Equal(t, byDirectOTLPResource, fromOCResource)
}
