// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json // import "go.opentelemetry.io/collector/pdata/internal/json"

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
)

func TestReadResource(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpresource.Resource
	}{
		{
			name:    "resource",
			jsonStr: `{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}],"dropped_attributes_count":1}`,
			want: &otlpresource.Resource{
				Attributes: []otlpcommon.KeyValue{
					{
						Key: "host.name",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_StringValue{
								StringValue: "testHost",
							},
						},
					},
				},
				DroppedAttributesCount: 1,
			},
		},
		{
			name:    "Unknown field",
			jsonStr: `{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}],"test":1}`,
			want: &otlpresource.Resource{
				Attributes: []otlpcommon.KeyValue{
					{
						Key: "host.name",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_StringValue{
								StringValue: "testHost",
							},
						},
					},
				},
				DroppedAttributesCount: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := &otlpresource.Resource{}
			ReadResource(iter, got)
			assert.NoError(t, iter.Error)
			assert.Equal(t, tt.want, got)
		})
	}
}
