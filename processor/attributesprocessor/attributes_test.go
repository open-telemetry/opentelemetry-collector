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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/span"
)

// Common structure for the
type testCase struct {
	name               string
	nodeName           string
	inputAttributes    map[string]*tracepb.AttributeValue
	expectedAttributes map[string]*tracepb.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, tp component.TraceProcessorOld) {
	t.Run(tt.name, func(t *testing.T) {
		traceData := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: tt.nodeName,
				},
			},
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.name},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.inputAttributes,
					},
				},
			},
		}

		assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
		require.Equal(t, consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: tt.nodeName,
				},
			},
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.name},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.expectedAttributes,
					},
				},
			},
		}, traceData)
	})
}

func TestAttributes_NilAttributes_Insert(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			nil,
			{
				Name:       &tracepb.TruncatableString{Value: "Nil Attributes"},
				Attributes: nil,
			},
			{
				Name:       &tracepb.TruncatableString{Value: "Empty Attributes"},
				Attributes: &tracepb.Span_Attributes{},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: nil,
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			nil,
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attributes"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"attribute1": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
					},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Empty Attributes"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"attribute1": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
					},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"attribute1": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
					},
				},
			},
		},
	}, traceData)
}

func TestAttributes_NilAttributes_Delete(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			nil,
			{
				Name:       &tracepb.TruncatableString{Value: "Nil Attributes"},
				Attributes: nil,
			},
			{
				Name:       &tracepb.TruncatableString{Value: "Empty Attributes"},
				Attributes: &tracepb.Span_Attributes{},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: nil,
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			nil,
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attributes"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Empty Attributes"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "Nil Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
		},
	}, traceData)
}

func TestAttributes_InsertValue(t *testing.T) {
	testCases := []testCase{
		// Ensure `attribute1` is set for spans with no attributes.
		{
			name:            "InsertEmptyAttributes",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		// Ensure `attribute1` is set.
		{
			name: "InsertKeyNoExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "InsertKeyExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_InsertFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure no attribute is inserted because because attributes do not exist.
		{
			name:               "InsertEmptyAttributes",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		// Ensure no attribute is inserted because because from_attribute `string_key` does not exist.
		{
			name: "InsertMissingFromAttribute",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"bob": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 1},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"bob": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 1},
				},
			},
		},
		// Ensure `string key` is set.
		{
			name: "InsertAttributeExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "InsertKeysExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"anotherkey": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 8892342},
				},
				"string key": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "here"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpdateValue(t *testing.T) {

	testCases := []testCase{
		// Ensure no changes to the span as there is no attributes map.
		{
			name:               "UpdateNoAttributes",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		// Ensure no changes to the span as the key does not exist.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "foo"}},
				},
			},
		},
		// Ensure the attribute `db.secret` is updated.
		{
			name: "UpdateAttributes",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"db.secret": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "password1234"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpdateFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure no changes to the span as there is no attributes map.
		{
			name:               "UpdateNoAttributes",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		// Ensure the attribute `boo` isn't updated because attribute `foo` isn't present in the span.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "over there"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "over there"}},
				},
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "there is a party over here"}},
				},
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "not here"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpsertValue(t *testing.T) {
	testCases := []testCase{
		// Ensure `region` is set for spans with no attributes.
		{
			name:            "UpsertNoAttributes",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "planet-earth"}},
				},
			},
		},
		// Ensure `region` is inserted for spans with some attributes(the key doesn't exist).
		{
			name: "UpsertAttributeNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "UpsertAttributeExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"mission": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "to mars"}},
				},
				"region": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "solar system"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_UpsertFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure `new_user_key` is not set for spans with no attributes.
		{
			name:               "UpsertEmptyAttributes",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			name: "UpsertFromAttributeNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				},
			},
		},
		// Ensure `new_user_key` is inserted for spans with attribute `user_key`.
		{
			name: "UpsertFromAttributeExistsInsert",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"user_key": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 2245},
				},
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "UpsertFromAttributeExistsUpdate",
			inputAttributes: map[string]*tracepb.AttributeValue{
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
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_Delete(t *testing.T) {
	testCases := []testCase{
		// Ensure the span contains no changes.
		{
			name:               "DeleteEmptyAttributes",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		// Ensure the span contains no changes because the key doesn't exist.
		{
			name: "DeleteAttributeNoExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"boo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
				}},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"boo": {Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}}},
			},
		},
		// Ensure `duplicate_key` is deleted for spans with the attribute set.
		{
			name: "DeleteAttributeExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"duplicate_key": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(3245.6)}},
				"original_key": {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(3245.6)}},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_FromAttributeNoChange(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "boo", Action: INSERT, FromAttribute: "boo"},
		{Key: "boo", Action: UPDATE, FromAttribute: "boo"},
		{Key: "boo", Action: UPSERT, FromAttribute: "boo"},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "FromAttributeNoChange"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"boo": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
						},
					},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "FromAttributeNoChange"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"bob": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
						},
					},
				},
			},
		},
	}

	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
	require.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "FromAttributeNoChange"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"boo": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
						},
					},
				},
			},
			{
				Name: &tracepb.TruncatableString{Value: "FromAttributeNoChange"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"bob": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "ghosts are scary"}},
						},
					},
				},
			},
		},
	}, traceData)
}

func TestAttributes_Ordering(t *testing.T) {
	testCases := []testCase{
		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. insert `svc.operation`: `default`
		// 3. delete `operation`.
		{
			name: "OrderingApplyAllSteps",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "OrderingOperationExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "OrderingSvcOperationExists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"foo": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "casper the friendly ghost"}},
				},
				"svc.operation": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "some value"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
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
			name: "OrderingBothAttributesExist",
			inputAttributes: map[string]*tracepb.AttributeValue{
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
			expectedAttributes: map[string]*tracepb.AttributeValue{
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

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_FilterSpans(t *testing.T) {
	testCases := []testCase{
		{
			name:            "apply processor",
			nodeName:        "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		{
			name:     "apply processor with different value for exclude property",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
		},
		{
			name:               "incorrect name for include property",
			nodeName:           "noname",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		{
			name:     "attribute match for exclude property",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
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
	oCfg.Include = &span.MatchProperties{
		Services:  []string{"svcA", "svcB.*"},
		MatchType: span.MatchTypeRegexp,
	}
	oCfg.Exclude = &span.MatchProperties{
		Attributes: []span.Attribute{
			{Key: "NoModification", Value: true},
		},
		MatchType: span.MatchTypeStrict,
	}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_FilterSpansByNameStrict(t *testing.T) {
	testCases := []testCase{
		{
			name:            "apply",
			nodeName:        "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		{
			name:     "apply",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
		},
		{
			name:               "incorrect_span_name",
			nodeName:           "svcB",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		{
			name:               "dont_apply",
			nodeName:           "svcB",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		{
			name:     "incorrect_span_name_with_attr",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
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
	oCfg.Include = &span.MatchProperties{
		SpanNames: []string{"apply", "dont_apply"},
		MatchType: span.MatchTypeStrict,
	}
	oCfg.Exclude = &span.MatchProperties{
		SpanNames: []string{"dont_apply"},
		MatchType: span.MatchTypeStrict,
	}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_FilterSpansByNameRegexp(t *testing.T) {
	testCases := []testCase{
		{
			name:            "apply_to_span_with_no_attrs",
			nodeName:        "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		{
			name:     "apply_to_span_with_attr",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
		},
		{
			name:               "incorrect_span_name",
			nodeName:           "svcB",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		{
			name:               "apply_dont_apply",
			nodeName:           "svcB",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
		{
			name:     "incorrect_span_name_with_attr",
			nodeName: "svcB",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
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
	oCfg.Include = &span.MatchProperties{
		SpanNames: []string{"^apply.*"},
		MatchType: span.MatchTypeRegexp,
	}
	oCfg.Exclude = &span.MatchProperties{
		SpanNames: []string{".*dont_apply$"},
		MatchType: span.MatchTypeRegexp,
	}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func TestAttributes_Hash(t *testing.T) {
	testCases := []testCase{
		{
			name: "String",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"user.email": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "john.doe@example.com"}},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"user.email": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "73ec53c4ba1747d485ae2a0d7bfafa6cda80a5a9"}},
				},
			},
		},
		{
			name: "Int",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"user.id": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 10},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"user.id": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "71aa908aff1548c8c6cdecf63545261584738a25"}},
				},
			},
		},
		{
			name: "Double",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"user.balance": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 99.1},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"user.balance": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "76429edab4855b03073f9429fd5d10313c28655e"}},
				},
			},
		},
		{
			name: "Bool",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"user.authenticated": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"user.authenticated": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bf8b4530d8d246dd74ac53a13471bba17941dff7"}},
				},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "user.email", Action: HASH},
		{Key: "user.id", Action: HASH},
		{Key: "user.balance", Action: HASH},
		{Key: "user.authenticated", Action: HASH},
	}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, tp)
	}
}

func BenchmarkAttributes_FilterSpansByName(b *testing.B) {
	testCases := []testCase{
		{
			name:            "apply_to_span_with_no_attrs",
			inputAttributes: map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		{
			name: "apply_to_span_with_attr",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
			expectedAttributes: map[string]*tracepb.AttributeValue{
				"attribute1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"NoModification": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: false},
				},
			},
		},
		{
			name:               "dont_apply",
			inputAttributes:    map[string]*tracepb.AttributeValue{},
			expectedAttributes: map[string]*tracepb.AttributeValue{},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}
	oCfg.Include = &span.MatchProperties{
		SpanNames: []string{"^apply.*"},
	}
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	require.Nil(b, err)
	require.NotNil(b, tp)

	for _, tt := range testCases {
		traceData := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: tt.nodeName,
				},
			},
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.name},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.inputAttributes,
					},
				},
			},
		}

		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				assert.NoError(b, tp.ConsumeTraceData(context.Background(), traceData))
			}
		})

		assert.Equal(b, consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{
					Name: tt.nodeName,
				},
			},
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.name},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.expectedAttributes,
					},
				},
			},
		}, traceData)
	}
}
