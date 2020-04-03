// Copyright 2020 OpenTelemetry Authors
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
	"strings"
	"testing"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestOcNodeResourceToInternal(t *testing.T) {
	resource := data.NewResource()
	ocNodeResourceToInternal(nil, nil, resource)
	assert.EqualValues(t, true, resource.IsNil())

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}
	ocNodeResourceToInternal(ocNode, ocResource, resource)
	assert.EqualValues(t, true, resource.IsNil())

	ocNode = generateOcNode()
	ocResource = generateOcResource()
	expectedAttrs := generateResourceWithOcNodeAndResource().Attributes()
	// We don't have type information in ocResource, so need to make int attr string
	expectedAttrs.Upsert(data.NewAttributeKeyValueString("resource-int-attr", "123"))
	ocNodeResourceToInternal(ocNode, ocResource, resource)
	assert.EqualValues(t, expectedAttrs.Sort(), resource.Attributes().Sort())

	// Make sure hard-coded fields override same-name values in Attributes.
	// To do that add Attributes with same-name.
	for i := 0; i < expectedAttrs.Len(); i++ {
		// Set all except "attr1" which is not a hard-coded field to some bogus values.

		if !strings.Contains(expectedAttrs.GetAttribute(i).Key(), "-attr") {
			ocNode.Attributes[expectedAttrs.GetAttribute(i).Key()] = "this will be overridden 1"
		}
	}
	ocResource.Labels[conventions.OCAttributeResourceType] = "this will be overridden 2"

	// Convert again.
	resource = data.NewResource()
	ocNodeResourceToInternal(ocNode, ocResource, resource)
	// And verify that same-name attributes were ignored.
	assert.EqualValues(t, expectedAttrs.Sort(), resource.Attributes().Sort())
}

func BenchmarkOcNodeResourceToInternal(b *testing.B) {
	ocNode := generateOcNode()
	ocResource := generateOcResource()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		resource := data.NewResource()
		ocNodeResourceToInternal(ocNode, ocResource, resource)
		if ocNode.Identifier.Pid != 123 {
			b.Fail()
		}
	}
}
