// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestReadScope(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpcommon.InstrumentationScope
	}{
		{
			name: "scope",
			jsonStr: `{
	"name": "name_value",
	"version": "version_value"
}`,
			want: &otlpcommon.InstrumentationScope{
				Name:    "name_value",
				Version: "version_value",
			},
		},
		{
			name: "with attributes",
			jsonStr: `{
	"name": "my_name",
	"version": "my_version",
	"attributes": [
		{
			"key":"string_key",
			"value":{ "stringValue": "value" }
		},
		{
			"key":"bool_key",
			"value":{ "boolValue": true }
		},
		{
			"key":"int_key",
			"value":{ "intValue": 314 }
		},
		{
			"key":"double_key",
			"value":{ "doubleValue": 3.14 }
		}
	],
	"dropped_attributes_count": 1
}`,
			want: &otlpcommon.InstrumentationScope{
				Name:    "my_name",
				Version: "my_version",
				Attributes: []otlpcommon.KeyValue{
					{
						Key: "string_key",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_StringValue{
								StringValue: "value",
							},
						},
					},
					{
						Key: "bool_key",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_BoolValue{
								BoolValue: true,
							},
						},
					},
					{
						Key: "int_key",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_IntValue{
								IntValue: 314,
							},
						},
					},
					{
						Key: "double_key",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_DoubleValue{
								DoubleValue: 3.14,
							},
						},
					},
				},
				DroppedAttributesCount: 1,
			},
		},
		{
			name: "unknown field",
			jsonStr: `{
	"name": "name_value",
	"unknown": "version"
}`,
			want: &otlpcommon.InstrumentationScope{
				Name: "name_value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := &otlpcommon.InstrumentationScope{}
			ReadScope(iter, got)
			require.NoError(t, iter.Error)
			assert.Equal(t, tt.want, got)
		})
	}
}
