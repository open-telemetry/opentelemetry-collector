// Copyright 2019, OpenTelemetry Authors
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

package attributesprocessor

import (
	"context"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

// Common structure for the
type testcase struct {
	name               string
	inputAttributes    map[string]*tracepb.AttributeValue
	expectedAttributes map[string]*tracepb.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testcase, tp processor.TraceProcessor) {
	t.Run(tt.name, func(t *testing.T) {
		traceData := consumerdata.TraceData{
			Spans: []*tracepb.Span{
				{Name: &tracepb.TruncatableString{Value: tt.name},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.inputAttributes,
					},
				},
			},
		}

		assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
		require.Equal(t, consumerdata.TraceData{
			Spans: []*tracepb.Span{{Name: &tracepb.TruncatableString{Value: tt.name},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: tt.expectedAttributes,
				},
			}},
		}, traceData)
	})
}

func TestAttributes_InsertValue(t *testing.T) {
	testcases := []testcase{
		// Ensure `attribute1` is set for spans with no attributes.
		{
			"InsertEmptyAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		// Ensure `attribute1` is set.
		{
			"InsertKeyNoExists",
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		// Ensures no insert is performed because the keys `attribute1` already exists.
		{
			"InsertKeyExists",
			map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_InsertFromAttribute(t *testing.T) {

	testcases := []testcase{
		// Ensure no attribute is inserted because because attributes do not exist.
		{
			"InsertEmptyAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{},
		},
		// Ensure no attribute is inserted because because from_attribute `string_key` does not exist.
		{
			"InsertMissingFromAttribute",
			map[string]*tracepb.AttributeValue{
				"bob": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 1},
				},
			},
			map[string]*tracepb.AttributeValue{
				"bob": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 1},
				},
			},
		},
		// Ensure `string key` is set.
		{
			"InsertAttributeExists",
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
			},
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
				"string key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
			},
		},
		// Ensures no insert is performed because the keys `string key` already exist.
		{
			"InsertKeysExists",
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
				"string key": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "here"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
				"string key": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "here"}},
				},
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "string key", Action: INSERT, FromAttribute: "anotherkey"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpdateValue(t *testing.T) {

	testcases := []testcase{
		// Ensure no changes to the span as there is no attributes map.
		{
			"UpdateNoAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{},
		},
		// Ensure no changes to the span as the key does not exist.
		{
			"UpdateKeyNoExist",
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
				},
			},
		},
		// Ensure the attribute `db.secret` is updated.
		{
			"UpdateAttributes",
			map[string]*tracepb.AttributeValue{
				"db.secret": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "password1234"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"db.secret": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "redacted"}},
				},
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "db.secret", Action: UPDATE, Value: "redacted"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpdateFromAttribute(t *testing.T) {

	testcases := []testcase{
		// Ensure no changes to the span as there is no attributes map.
		{
			"UpdateNoAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{},
		},
		// Ensure the attribute `boo` isn't updated because attribute `foo` isn't present in the span.
		{
			"UpdateKeyNoExist",
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			"UpdateKeyNoExist",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "over there"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "over there"}},
				},
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			"UpdateKeyNoExist",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "there is a party over here"}},
				},
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "not here"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "there is a party over here"}},
				},
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "there is a party over here"}},
				},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "boo", Action: UPDATE, FromAttribute: "foo"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpsertValue(t *testing.T) {
	testcases := []testcase{
		// Ensure `region` is set for spans with no attributes.
		{
			"UpsertNoAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "planet-earth"}},
				},
			},
		},
		// Ensure `region` is inserted for spans with some attributes(the key doesn't exist).
		{
			"UpsertAttributeNoExist",
			map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "planet-earth"}},
				},
			},
		},
		/// Ensure `region` is updated for spans with the attribute key `region`.
		{
			"UpsertAttributeExists",
			map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "solar system"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "planet-earth"}},
				},
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "region", Action: UPSERT, Value: "planet-earth"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}
func TestAttributes_UpsertFromAttribute(t *testing.T) {

	testcases := []testcase{
		// Ensure `new_user_key` is not set for spans with no attributes.
		{
			"UpsertEmptyAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			"UpsertFromAttributeNoExist",
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				},
			},
		},
		// Ensure `new_user_key` is inserted for spans with attribute `user_key`.
		{
			"UpsertFromAttributeExistsInsert",
			map[string]*tracepb.AttributeValue{
				"user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"new_user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
		},
		// Ensure `new_user_key` is updated for spans with attribute `user_key`.
		{
			"UpsertFromAttributeExistsUpdate",
			map[string]*tracepb.AttributeValue{
				"user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"new_user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 5422},
				},
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"new_user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "new_user_key", Action: UPSERT, FromAttribute: "user_key"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_Delete(t *testing.T) {
	testcases := []testcase{
		// Ensure the span contains no changes.
		{
			"DeleteEmptyAttributes",
			map[string]*tracepb.AttributeValue{},
			map[string]*tracepb.AttributeValue{},
		},
		// Ensure the span contains no changes because the key doesn't exist.
		{
			"DeleteAttributeNoExist",
			map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				}},
			map[string]*tracepb.AttributeValue{
				"boo": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}}},
			},
		},
		// Ensure `duplicate_key` is deleted for spans with the attribute set.
		{
			"DeleteAttributeExists",
			map[string]*tracepb.AttributeValue{
				"duplicate_key": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(3245.6)}},
				"original_key": {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(3245.6)}},
			},
			map[string]*tracepb.AttributeValue{
				"original_key": {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(3245.6)}},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "duplicate_key", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_Ordering(t *testing.T) {
	testcases := []struct {
		name               string
		inputAttributes    map[string]*tracepb.AttributeValue
		expectedAttributes map[string]*tracepb.AttributeValue
	}{
		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. insert `svc.operation`: `default`
		// 3. delete `operation`.
		{
			"OrderingApplyAllSteps",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "default"}},
				},
			},
		},
		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. insert `svc.operation`: `arithmetic`
		// 3. delete `operation`.
		{
			"OrderingOperationExists",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
			},
		},

		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. update `svc.operation` to `default`
		// 3. delete `operation`.
		{
			"OrderingSvcOperationExists",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "some value"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "default"}},
				},
			},
		},

		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. update `svc.operation` to `arithmetic`
		// 3. delete `operation`.
		{
			"OrderingBothAttributesExist",
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "add"}},
				},
			},
			map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "operation", Action: INSERT, Value: "default"},
		{Key: "svc.operation", Action: UPSERT, FromAttribute: "operation"},
		{Key: "operation", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testcases {
		runIndividualTestCase(t, tt, tp)
	}
}
