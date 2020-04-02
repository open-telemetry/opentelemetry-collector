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
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceToOC(t *testing.T) {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocAttributes := map[string]string{
		"str1": "text",
		"int2": "123",
	}

	attrs := map[string]data.AttributeValue{
		conventions.OCAttributeProcessStartTime: data.NewAttributeValueString("2020-02-11T20:26:00Z"),
		conventions.AttributeHostHostname:       data.NewAttributeValueString("host1"),
		conventions.OCAttributeProcessID:        data.NewAttributeValueString("123"),
		conventions.AttributeLibraryVersion:     data.NewAttributeValueString("v2.0.1"),
		conventions.OCAttributeExporterVersion:  data.NewAttributeValueString("v1.2.0"),
		conventions.AttributeLibraryLanguage:    data.NewAttributeValueString("CPP"),
		conventions.OCAttributeResourceType:     data.NewAttributeValueString("good-resource"),
		"str1":                                  data.NewAttributeValueString("text"),
		"int2":                                  data.NewAttributeValueInt(123),
	}
	resource := data.NewResource()
	resource.InitEmpty()
	resource.Attributes().InitFromMap(attrs)

	emptyResource := data.NewResource()
	emptyResource.InitEmpty()

	tests := []struct {
		name       string
		resource   data.Resource
		ocNode     *occommon.Node
		ocResource *ocresource.Resource
	}{
		{
			name:       "nil",
			resource:   data.NewResource(),
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
			name:     "with-attributes",
			resource: resource,
			ocNode: &occommon.Node{
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
			},
			ocResource: &ocresource.Resource{
				Type:   "good-resource",
				Labels: ocAttributes,
			},
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
