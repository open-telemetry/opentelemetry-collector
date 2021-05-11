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
	"sort"
	"testing"
	"unsafe"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpresource "go.opentelemetry.io/collector/internal/data/protogen/resource/v1"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func TestCreateNodeAndResourceConversion(t *testing.T) {
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

	fromOCResource := protoResource(mdFromOC.ResourceMetrics().At(0).Resource())
	byDirectOTLPResource := protoResource(createNodeAndResourcePdata(job, instance, scheme))

	if diff := cmp.Diff(byDirectOTLPResource, fromOCResource, protocmp.Transform()); diff != "" {
		t.Fatalf("Resource mismatch: got: - want: +\n%s", diff)
	}
}

// Unfortunately pdata doesn't expose a way for us to retrieve the underlying resource,
// yet we need to compare the resources which are hidden by an unexported value.
func protoResource(presource pdata.Resource) *otlpresource.Resource {
	type extract struct {
		orig *otlpresource.Resource
	}
	extracted := (*extract)(unsafe.Pointer(&presource))
	if extracted == nil {
		return nil
	}
	// Ensure that the attributes are sorted so that we can properly compare the raw values.
	resource := extracted.orig
	sort.Slice(resource.Attributes, func(i, j int) bool {
		return resource.Attributes[i].Key < resource.Attributes[j].Key
	})
	return resource
}
