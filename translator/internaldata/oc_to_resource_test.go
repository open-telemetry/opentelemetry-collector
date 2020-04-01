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
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
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

	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocNode = &occommon.Node{
		Identifier: &occommon.ProcessIdentifier{
			HostName:       "host1",
			Pid:            123,
			StartTimestamp: ts,
		},
		LibraryInfo: &occommon.LibraryInfo{
			Language:           occommon.LibraryInfo_CPP,
			ExporterVersion:    "v1.2.0",
			CoreLibraryVersion: "v2.0.1",
		},
		ServiceInfo: &occommon.ServiceInfo{
			Name: "svcA",
		},
		Attributes: map[string]string{
			"node-attr": "val1",
		},
	}
	ocResource = &ocresource.Resource{
		Type: "good-resource",
		Labels: map[string]string{
			"resource-attr": "val2",
		},
	}
	expectedAttrs := data.NewAttributeMap().InitFromMap(map[string]data.AttributeValue{
		conventions.AttributeHostHostname:       data.NewAttributeValueString("host1"),
		conventions.OCAttributeProcessID:        data.NewAttributeValueInt(123),
		conventions.OCAttributeProcessStartTime: data.NewAttributeValueString("2020-02-11T20:26:00Z"),
		conventions.AttributeLibraryLanguage:    data.NewAttributeValueString("CPP"),
		conventions.OCAttributeExporterVersion:  data.NewAttributeValueString("v1.2.0"),
		conventions.AttributeLibraryVersion:     data.NewAttributeValueString("v2.0.1"),
		conventions.AttributeServiceName:        data.NewAttributeValueString("svcA"),
		"node-attr":                             data.NewAttributeValueString("val1"),
		conventions.OCAttributeResourceType:     data.NewAttributeValueString("good-resource"),
		"resource-attr":                         data.NewAttributeValueString("val2"),
	})

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
