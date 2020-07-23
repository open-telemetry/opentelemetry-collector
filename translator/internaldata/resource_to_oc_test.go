// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internaldata

import (
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestResourceToOC(t *testing.T) {
	emptyResource := pdata.NewResource()
	emptyResource.InitEmpty()

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
			ocNode:     &occommon.Node{},
			ocResource: &ocresource.Resource{},
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

func TestAttributeValueToString(t *testing.T) {
	assert.EqualValues(t, "", attributeValueToString(pdata.NewAttributeValueNull(), false))
	assert.EqualValues(t, "abc", attributeValueToString(pdata.NewAttributeValueString("abc"), false))
	assert.EqualValues(t, `"abc"`, attributeValueToString(pdata.NewAttributeValueString("abc"), true))
	assert.EqualValues(t, "123", attributeValueToString(pdata.NewAttributeValueInt(123), false))
	assert.EqualValues(t, "1.23", attributeValueToString(pdata.NewAttributeValueDouble(1.23), false))
	assert.EqualValues(t, "true", attributeValueToString(pdata.NewAttributeValueBool(true), false))

	v := pdata.NewAttributeValueMap()
	v.MapVal().InsertString(`a"\`, `b"\`)
	v.MapVal().InsertInt("c", 123)
	v.MapVal().Insert("d", pdata.NewAttributeValueNull())
	v.MapVal().Insert("e", v)
	assert.EqualValues(t, `{"a\"\\":"b\"\\","c":123,"d":null,"e":{"a\"\\":"b\"\\","c":123,"d":null}}`, attributeValueToString(v, false))
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
