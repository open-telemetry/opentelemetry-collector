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

package zipkinv1

import (
	"fmt"
	"strconv"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestAttribToStatusCode(t *testing.T) {
	_, atoiError := strconv.Atoi("nan")

	tests := []struct {
		name string
		attr *tracepb.AttributeValue
		code int32
		err  error
	}{
		{
			name: "nil",
			attr: nil,
			code: 0,
			err:  fmt.Errorf("nil attribute"),
		},

		{
			name: "valid-int-code",
			attr: &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_IntValue{IntValue: int64(0)},
			},
			code: 0,
			err:  nil,
		},

		{
			name: "invalid-int-code",
			attr: &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_IntValue{IntValue: int64(1 << 32)},
			},
			code: 0,
			err:  fmt.Errorf("outside of the int32 range"),
		},

		{
			name: "valid-string-code",
			attr: &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "200"}},
			},
			code: 200,
			err:  nil,
		},

		{
			name: "invalid-string-code",
			attr: &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "nan"}},
			},
			code: 0,
			err:  atoiError,
		},

		{
			name: "bool-code",
			attr: &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
			},
			code: 0,
			err:  fmt.Errorf("invalid attribute type"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := attribToStatusCode(test.attr)
			assert.Equal(t, test.code, got)
			assert.Equal(t, test.err, err)
		})
	}
}

func TestStatusCodeMapperCases(t *testing.T) {
	tests := []struct {
		name       string
		expected   *tracepb.Status
		attributes map[string]string
	}{
		{
			name:     "no relevant attributes",
			expected: nil,
			attributes: map[string]string{
				"not.relevant": "data",
			},
		},

		{
			name:     "http: 500",
			expected: &tracepb.Status{Code: 13},
			attributes: map[string]string{
				"http.status_code": "500",
			},
		},

		{
			name:     "http: message only, nil",
			expected: nil,
			attributes: map[string]string{
				"http.status_message": "something",
			},
		},

		{
			name:     "http: 500",
			expected: &tracepb.Status{Code: 13, Message: "a message"},
			attributes: map[string]string{
				"http.status_code":    "500",
				"http.status_message": "a message",
			},
		},

		{
			name:     "http: 500, with error attribute",
			expected: &tracepb.Status{Code: 13},
			attributes: map[string]string{
				"http.status_code": "500",
				"error":            "an error occurred",
			},
		},

		{
			name:     "oc: internal",
			expected: &tracepb.Status{Code: 13, Message: "a description"},
			attributes: map[string]string{
				"census.status_code":        "13",
				"census.status_description": "a description",
			},
		},

		{
			name:     "oc: description and error",
			expected: &tracepb.Status{Code: 13, Message: "a description"},
			attributes: map[string]string{
				"opencensus.status_description": "a description",
				"error":                         "INTERNAL",
			},
		},

		{
			name:     "oc: error only",
			expected: &tracepb.Status{Code: 13, Message: ""},
			attributes: map[string]string{
				"error": "INTERNAL",
			},
		},

		{
			name:     "oc: empty error tag",
			expected: nil,
			attributes: map[string]string{
				"error": "",
			},
		},

		{
			name:     "oc: description only, no status",
			expected: nil,
			attributes: map[string]string{
				"opencensus.status_description": "a description",
			},
		},

		{
			name:     "oc: priority over http",
			expected: &tracepb.Status{Code: 4, Message: "deadline expired"},
			attributes: map[string]string{
				"census.status_description": "deadline expired",
				"census.status_code":        "4",

				"http.status_message": "a description",
				"http.status_code":    "500",
			},
		},

		{
			name:     "error: valid oc status priority over http",
			expected: &tracepb.Status{Code: 4},
			attributes: map[string]string{
				"error": "DEADLINE_EXCEEDED",

				"http.status_message": "a description",
				"http.status_code":    "500",
			},
		},

		{
			name:     "error: invalid oc status uses http",
			expected: &tracepb.Status{Code: 13, Message: "a description"},
			attributes: map[string]string{
				"error": "123",

				"http.status_message": "a description",
				"http.status_code":    "500",
			},
		},

		{
			name:     "error only: string description",
			expected: &tracepb.Status{Code: 2},
			attributes: map[string]string{
				"error": "a description",
			},
		},

		{
			name:     "error only: true",
			expected: &tracepb.Status{Code: 2},
			attributes: map[string]string{
				"error": "true",
			},
		},

		{
			name:     "error only: false",
			expected: &tracepb.Status{Code: 2},
			attributes: map[string]string{
				"error": "false",
			},
		},

		{
			name:     "error only: 1",
			expected: &tracepb.Status{Code: 2},
			attributes: map[string]string{
				"error": "1",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attributes := attributesFromMap(test.attributes)

			sMapper := &statusMapper{}
			for k, v := range attributes {
				sMapper.fromAttribute(k, v)
			}

			got := sMapper.ocStatus()
			assert.EqualValues(t, test.expected, got)
		})
	}
}

func attributesFromMap(mapValues map[string]string) map[string]*tracepb.AttributeValue {
	res := map[string]*tracepb.AttributeValue{}

	for k, v := range mapValues {
		pbAttrib := parseAnnotationValue(v, false)
		res[k] = pbAttrib
	}
	return res
}
