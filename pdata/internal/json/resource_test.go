// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
