// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal/json"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
	otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
	"go.opentelemetry.io/collector/pdata/internal/json"
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
			iter := json.BorrowIterator([]byte(tt.jsonStr))
			defer json.ReturnIterator(iter)
			got := &otlpresource.Resource{}
			UnmarshalJSONIterResource(NewResource(got, nil), iter)
			require.NoError(t, iter.Error())
			assert.Equal(t, tt.want, got)
		})
	}
}
