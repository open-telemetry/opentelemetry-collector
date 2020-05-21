// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkin

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
