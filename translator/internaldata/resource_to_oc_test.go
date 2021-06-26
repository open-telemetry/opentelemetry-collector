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

package internaldata

import (
	"strconv"
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/resource/resourcekeys"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/occonventions"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestResourceToOC(t *testing.T) {
	emptyResource := pdata.NewResource()

	ocNode := generateOcNode()
	ocResource := generateOcResource()
	// We don't differentiate between Node.Attributes and Resource when converting,
	// and put everything in Resource.
	ocResource.Labels["node-str-attr"] = "node-str-attr-val"
	ocNode.Attributes = nil

	tests := []struct {
		name       string
		resource   pdata.Resource
		ocNode     *occommon.Node
		ocResource *ocresource.Resource
	}{
		{
			name:       "nil",
			resource:   pdata.NewResource(),
			ocNode:     nil,
			ocResource: nil,
		},

		{
			name:       "empty",
			resource:   emptyResource,
			ocNode:     nil,
			ocResource: nil,
		},

		{
			name:       "with-attributes",
			resource:   generateResourceWithOcNodeAndResource(),
			ocNode:     ocNode,
			ocResource: ocResource,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ocNode, ocResource := internalResourceToOC(test.resource)
			assert.EqualValues(t, test.ocNode, ocNode)
			assert.EqualValues(t, test.ocResource, ocResource)
		})
	}
}

func TestContainerResourceToOC(t *testing.T) {
	resource := pdata.NewResource()
	resource.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.AttributeK8sCluster:            pdata.NewAttributeValueString("cluster1"),
		conventions.AttributeK8sPod:                pdata.NewAttributeValueString("pod1"),
		conventions.AttributeK8sNamespace:          pdata.NewAttributeValueString("namespace1"),
		conventions.AttributeContainerName:         pdata.NewAttributeValueString("container-name1"),
		conventions.AttributeCloudAccount:          pdata.NewAttributeValueString("proj1"),
		conventions.AttributeCloudAvailabilityZone: pdata.NewAttributeValueString("zone1"),
	})

	want := &ocresource.Resource{
		Type: resourcekeys.ContainerType, // Inferred type
		Labels: map[string]string{
			resourcekeys.K8SKeyClusterName:   "cluster1",
			resourcekeys.K8SKeyPodName:       "pod1",
			resourcekeys.K8SKeyNamespaceName: "namespace1",
			resourcekeys.ContainerKeyName:    "container-name1",
			resourcekeys.CloudKeyAccountID:   "proj1",
			resourcekeys.CloudKeyZone:        "zone1",
		},
	}

	_, ocResource := internalResourceToOC(resource)
	if diff := cmp.Diff(want, ocResource, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected difference:\n%v", diff)
	}

	// Also test that the explicit resource type is preserved if present
	resource.Attributes().InsertString(occonventions.AttributeResourceType, "other-type")
	want.Type = "other-type"

	_, ocResource = internalResourceToOC(resource)
	if diff := cmp.Diff(want, ocResource, protocmp.Transform()); diff != "" {
		t.Errorf("Unexpected difference:\n%v", diff)
	}
}

func TestInferResourceType(t *testing.T) {
	tests := []struct {
		name             string
		labels           map[string]string
		wantResourceType string
		wantOk           bool
	}{
		{
			name:   "empty labels",
			labels: nil,
			wantOk: false,
		},
		{
			name: "container",
			labels: map[string]string{
				conventions.AttributeK8sCluster:            "cluster1",
				conventions.AttributeK8sPod:                "pod1",
				conventions.AttributeK8sNamespace:          "namespace1",
				conventions.AttributeContainerName:         "container-name1",
				conventions.AttributeCloudAccount:          "proj1",
				conventions.AttributeCloudAvailabilityZone: "zone1",
			},
			wantResourceType: resourcekeys.ContainerType,
			wantOk:           true,
		},
		{
			name: "pod",
			labels: map[string]string{
				conventions.AttributeK8sCluster:            "cluster1",
				conventions.AttributeK8sPod:                "pod1",
				conventions.AttributeK8sNamespace:          "namespace1",
				conventions.AttributeCloudAvailabilityZone: "zone1",
			},
			wantResourceType: resourcekeys.K8SType,
			wantOk:           true,
		},
		{
			name: "host",
			labels: map[string]string{
				conventions.AttributeK8sCluster:            "cluster1",
				conventions.AttributeCloudAvailabilityZone: "zone1",
				conventions.AttributeHostName:              "node1",
			},
			wantResourceType: resourcekeys.HostType,
			wantOk:           true,
		},
		{
			name: "gce",
			labels: map[string]string{
				conventions.AttributeCloudProvider:         "gcp",
				conventions.AttributeHostID:                "inst1",
				conventions.AttributeCloudAvailabilityZone: "zone1",
			},
			wantResourceType: resourcekeys.CloudType,
			wantOk:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resourceType, ok := inferResourceType(tc.labels)
			if tc.wantOk {
				assert.True(t, ok)
				assert.Equal(t, tc.wantResourceType, resourceType)
			} else {
				assert.False(t, ok)
				assert.Equal(t, "", resourceType)
			}
		})
	}
}

func TestResourceToOCAndBack(t *testing.T) {
	tests := []goldendataset.PICTInputResource{
		goldendataset.ResourceEmpty,
		goldendataset.ResourceVMOnPrem,
		goldendataset.ResourceVMCloud,
		goldendataset.ResourceK8sOnPrem,
		goldendataset.ResourceK8sCloud,
		goldendataset.ResourceFaas,
		goldendataset.ResourceExec,
	}
	for _, test := range tests {
		t.Run(string(test), func(t *testing.T) {
			traces := pdata.NewTraces()
			goldendataset.GenerateResource(test).CopyTo(traces.ResourceSpans().AppendEmpty().Resource())
			expected := traces.ResourceSpans().At(0).Resource()
			ocNode, ocResource := internalResourceToOC(expected)
			actual := pdata.NewResource()
			ocNodeResourceToInternal(ocNode, ocResource, actual)
			// Remove opencensus resource type from actual. This will be added during translation.
			actual.Attributes().Delete(occonventions.AttributeResourceType)
			assert.Equal(t, expected.Attributes().Len(), actual.Attributes().Len())
			expected.Attributes().Range(func(k string, v pdata.AttributeValue) bool {
				a, ok := actual.Attributes().Get(k)
				assert.True(t, ok)
				switch v.Type() {
				case pdata.AttributeValueTypeInt:
					// conventions.AttributeProcessID is special because we preserve the type for this.
					if k == conventions.AttributeProcessID {
						assert.Equal(t, v.IntVal(), a.IntVal())
					} else {
						assert.Equal(t, strconv.FormatInt(v.IntVal(), 10), a.StringVal())
					}
				case pdata.AttributeValueTypeMap, pdata.AttributeValueTypeArray:
					assert.Equal(t, a, a)
				default:
					assert.Equal(t, v, a)
				}
				return true
			})
		})
	}
}

func BenchmarkInternalResourceToOC(b *testing.B) {
	resource := generateResourceWithOcNodeAndResource()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ocNode, _ := internalResourceToOC(resource)
		if ocNode.Identifier.Pid != 123 {
			b.Fail()
		}
	}
}

func BenchmarkOcResourceNodeMarshal(b *testing.B) {
	oc := &agenttracepb.ExportTraceServiceRequest{
		Node:     generateOcNode(),
		Spans:    nil,
		Resource: generateOcResource(),
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := proto.Marshal(oc); err != nil {
			b.Fail()
		}
	}
}
