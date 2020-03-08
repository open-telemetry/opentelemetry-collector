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

func TestResourceToOC(t *testing.T) {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	otlpAttributes := []*otlpcommon.AttributeKeyValue{
		{
			Key:         conventions.OCAttributeResourceType,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "good-resource",
		},
		{
			Key:         conventions.OCAttributeProcessStartTime,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "2020-02-11T20:26:00Z",
		},
		{
			Key:         conventions.AttributeHostHostname,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "host1",
		},
		{
			Key:         conventions.OCAttributeProcessID,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "123",
		},
		{
			Key:         conventions.AttributeLibraryVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v2.0.1",
		},
		{
			Key:         conventions.OCAttributeExporterVersion,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "v1.2.0",
		},
		{
			Key:         conventions.AttributeLibraryLanguage,
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "CPP",
		},
		{
			Key:         "str1",
			Type:        otlpcommon.AttributeKeyValue_STRING,
			StringValue: "text",
		},
		{
			Key:      "int2",
			Type:     otlpcommon.AttributeKeyValue_INT,
			IntValue: 123,
		},
	}

	ocAttributes := map[string]string{
		"str1": "text",
		"int2": "123",
	}

	tests := []struct {
		name         string
		otlpResource *otlpresource.Resource
		ocNode       *occommon.Node
		ocResource   *ocresource.Resource
	}{
		{
			name:         "nil",
			otlpResource: nil,
			ocNode:       nil,
			ocResource:   nil,
		},

		{
			name:         "empty",
			otlpResource: &otlpresource.Resource{},
			ocNode:       &occommon.Node{},
			ocResource:   &ocresource.Resource{},
		},

		{
			name: "with-attributes",
			otlpResource: &otlpresource.Resource{
				Attributes: otlpAttributes,
			},
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
			ocNode, ocResource := ResourceToOC(test.otlpResource)
			assert.EqualValues(t, test.ocNode, ocNode)
			assert.EqualValues(t, test.ocResource, ocResource)
		})
	}
}
