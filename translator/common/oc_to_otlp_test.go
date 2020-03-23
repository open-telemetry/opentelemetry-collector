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

package common

import (
	"strings"
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestOcNodeResourceToOtlp(t *testing.T) {
	otlpResource := OCNodeResourceToOtlp(nil, nil)
	assert.EqualValues(t, &otlpresource.Resource{
		Attributes: nil,
	}, otlpResource)

	node := &occommon.Node{}
	resource := &ocresource.Resource{}
	otlpResource = OCNodeResourceToOtlp(node, resource)
	assert.EqualValues(t, &otlpresource.Resource{
		Attributes: nil,
	}, otlpResource)

	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	node = &occommon.Node{
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
	resource = &ocresource.Resource{
		Type: "good-resource",
		Labels: map[string]string{
			"resource-attr": "val2",
		},
	}
	otlpResource = OCNodeResourceToOtlp(node, resource)

	expectedAttrs := map[string]string{
		conventions.AttributeHostHostname:       "host1",
		conventions.OCAttributeProcessID:        "123",
		conventions.OCAttributeProcessStartTime: "2020-02-11T20:26:00Z",
		conventions.AttributeLibraryLanguage:    "CPP",
		conventions.OCAttributeExporterVersion:  "v1.2.0",
		conventions.AttributeLibraryVersion:     "v2.0.1",
		conventions.AttributeServiceName:        "svcA",
		"node-attr":                             "val1",
		conventions.OCAttributeResourceType:     "good-resource",
		"resource-attr":                         "val2",
	}

	assert.EqualValues(t, len(expectedAttrs), len(otlpResource.Attributes))
	for k, v := range expectedAttrs {
		assertHasAttrStr(t, otlpResource.Attributes, k, v)
	}

	// Make sure hard-coded fields override same-name values in Attributes.
	// To do that add Attributes with same-name.
	for k := range expectedAttrs {
		// Set all except "attr1" which is not a hard-coded field to some bogus values.

		if strings.Index(k, "-attr") < 0 {
			node.Attributes[k] = "this will be overridden 1"
		}
	}
	resource.Labels[conventions.OCAttributeResourceType] = "this will be overridden 2"

	// Convert again.
	otlpResource = OCNodeResourceToOtlp(node, resource)

	// And verify that same-name attributes were ignored.
	assert.EqualValues(t, len(expectedAttrs), len(otlpResource.Attributes))
	for k, v := range expectedAttrs {
		assertHasAttrStr(t, otlpResource.Attributes, k, v)
	}
}

func assertHasAttrStr(t *testing.T, attrs []*otlpcommon.AttributeKeyValue, key string, val string) {
	assertHasAttrVal(t, attrs, key, &otlpcommon.AttributeKeyValue{
		Key:         key,
		Type:        otlpcommon.AttributeKeyValue_STRING,
		StringValue: val,
	})
}

func assertHasAttrVal(t *testing.T, attrs []*otlpcommon.AttributeKeyValue, key string, val *otlpcommon.AttributeKeyValue) {
	found := false
	for _, attr := range attrs {
		if attr != nil && attr.Key == key {
			assert.EqualValues(t, false, found, "Duplicate key "+key)
			assert.EqualValues(t, val, attr)
			found = true
		}
	}
	assert.EqualValues(t, true, found, "Cannot find key "+key)
}

func TestAttrMapToOtlp(t *testing.T) {
	ocAttrs := map[string]string{}

	otlpAttrs := attrMapToOtlp(ocAttrs)
	assert.True(t, otlpAttrs == nil)

	ocAttrs = map[string]string{"abc": "def"}
	otlpAttrs = attrMapToOtlp(ocAttrs)
	assert.EqualValues(t, []*otlpcommon.AttributeKeyValue{
		{
			Key:         "abc",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "def",
		},
	}, otlpAttrs)
}
