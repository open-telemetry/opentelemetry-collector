// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestReadArray(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpcommon.ArrayValue
	}{
		{
			name: "values",
			jsonStr: `{"values":[{
"stringValue":"12312"
}]}`,
			want: &otlpcommon.ArrayValue{
				Values: []otlpcommon.AnyValue{
					{
						Value: &otlpcommon.AnyValue_StringValue{
							StringValue: "12312",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := readArray(iter)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestReadKvlistValue(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		want    *otlpcommon.KeyValueList
	}{
		{
			name: "values",
			jsonStr: `{"values":[{
"key":"testKey",
"value":{
"stringValue": "testValue"
}
}]}`,
			want: &otlpcommon.KeyValueList{
				Values: []otlpcommon.KeyValue{
					{
						Key: "testKey",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_StringValue{
								StringValue: "testValue",
							},
						},
					},
				},
			},
		},
		{
			name: "boolValue",
			jsonStr: `{"values":[{
"key":"testKey",
"value":{
"boolValue": true
}
}]}`,
			want: &otlpcommon.KeyValueList{
				Values: []otlpcommon.KeyValue{
					{
						Key: "testKey",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_BoolValue{
								BoolValue: true,
							},
						},
					},
				},
			},
		},
		{
			name: "intValue",
			jsonStr: `{"values":[{
"key":"testKey",
"value":{
"intValue": 1
}
}]}`,
			want: &otlpcommon.KeyValueList{
				Values: []otlpcommon.KeyValue{
					{
						Key: "testKey",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_IntValue{
								IntValue: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "doubleValue",
			jsonStr: `{"values":[{
"key":"testKey",
"value":{
"doubleValue": 1.3
}
}]}`,
			want: &otlpcommon.KeyValueList{
				Values: []otlpcommon.KeyValue{
					{
						Key: "testKey",
						Value: otlpcommon.AnyValue{
							Value: &otlpcommon.AnyValue_DoubleValue{
								DoubleValue: 1.3,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iter := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.jsonStr))
			defer jsoniter.ConfigFastest.ReturnIterator(iter)
			got := readKvlistValue(iter)
			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestReadAttributeUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := ReadAttribute(iter)
	//  unknown fields should not be an error
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, otlpcommon.KeyValue{}, value)
}

func TestReadAttributeValueUnknownField(t *testing.T) {
	// Key after value, to check that we correctly continue to process.
	jsonStr := `{"value": {"unknown": {"extra":""}}, "key":"test"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := ReadAttribute(iter)
	//  unknown fields should not be an error
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, otlpcommon.KeyValue{Key: "test"}, value)
}

func TestReadValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := &otlpcommon.AnyValue{}
	ReadValue(iter, value)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpcommon.AnyValue{}, value)
}

func TestReadValueInvliadBytesValue(t *testing.T) {
	jsonStr := `{"bytesValue": "--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)

	ReadValue(iter, &otlpcommon.AnyValue{})
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "base64")
	}
}

func TestReadArrayUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readArray(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpcommon.ArrayValue{}, value)
}

func TestReadKvlistValueUnknownField(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	value := readKvlistValue(iter)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpcommon.KeyValueList{}, value)
}

func TestReadArrayValueInvalidArrayValue(t *testing.T) {
	jsonStr := `{"arrayValue": {"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)

	value := &otlpcommon.AnyValue{}
	ReadValue(iter, value)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpcommon.AnyValue{
		Value: &otlpcommon.AnyValue_ArrayValue{
			ArrayValue: &otlpcommon.ArrayValue{},
		},
	}, value)
}

func TestReadKvlistValueInvalidArrayValue(t *testing.T) {
	jsonStr := `{"kvlistValue": {"extra":""}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)

	value := &otlpcommon.AnyValue{}
	ReadValue(iter, value)
	assert.NoError(t, iter.Error)
	assert.EqualValues(t, &otlpcommon.AnyValue{
		Value: &otlpcommon.AnyValue_KvlistValue{
			KvlistValue: &otlpcommon.KeyValueList{},
		},
	}, value)
}
