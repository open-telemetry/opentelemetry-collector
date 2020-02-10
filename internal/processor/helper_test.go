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

package processor

import (
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func TestHelper_AttributeValue(t *testing.T) {
	val, err := AttributeValue(123)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(123)}}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(234.129312)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)}}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(true)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
	}, val)
	assert.Nil(t, err)

	val, err = AttributeValue("bob the builder")
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob the builder"}},
	}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(nil)
	assert.Nil(t, val)
	assert.Equal(t, "error unsupported value type \"<nil>\"", err.Error())

	val, err = AttributeValue(t)
	assert.Nil(t, val)
	assert.Equal(t, "error unsupported value type \"*testing.T\"", err.Error())
}
