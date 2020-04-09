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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterspan"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

// Common structure for the
type testCase struct {
	name               string
	serviceName        string
	inputAttributes    map[string]data.AttributeValue
	expectedAttributes map[string]data.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, tp component.TraceProcessor) {
	t.Run(tt.name, func(t *testing.T) {
		td := generateTraceData(tt.serviceName, tt.name, tt.inputAttributes)
		assert.NoError(t, tp.ConsumeTrace(context.Background(), td))
		// Ensure that the modified `td` has the attributes sorted:
		sortAttributes(td)
		require.Equal(t, generateTraceData(tt.serviceName, tt.name, tt.expectedAttributes), td)
	})
}

func generateTraceData(serviceName, spanName string, attrs map[string]data.AttributeValue) data.TraceData {
	td := data.NewTraceData()
	td.ResourceSpans().Resize(1)
	rs := td.ResourceSpans().At(0)
	if serviceName != "" {
		rs.Resource().InitEmpty()
		rs.Resource().Attributes().UpsertString(conventions.AttributeServiceName, serviceName)
	}
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	spans := ils.Spans()
	spans.Resize(1)
	spans.At(0).SetName(spanName)
	spans.At(0).Attributes().InitFromMap(attrs).Sort()
	return td
}

func sortAttributes(td data.TraceData) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		if rs.IsNil() {
			continue
		}
		if !rs.Resource().IsNil() {
			rs.Resource().Attributes().Sort()
		}
		ilss := rss.At(i).InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				s := spans.At(k)
				if !s.IsNil() {
					s.Attributes().Sort()
				}
			}
		}
	}
}

// TestSpanProcessor_Values tests all possible value types.
func TestSpanProcessor_NilEmptyData(t *testing.T) {
	type nilEmptyTestCase struct {
		name   string
		input  data.TraceData
		output data.TraceData
	}
	// TODO: Add test for "nil" Span/Attributes. This needs support from data slices to allow to construct that.
	testCases := []nilEmptyTestCase{
		{
			name:   "empty",
			input:  testdata.GenerateTraceDataEmpty(),
			output: testdata.GenerateTraceDataEmpty(),
		},
		{
			name:   "one-empty-resource-spans",
			input:  testdata.GenerateTraceDataOneEmptyResourceSpans(),
			output: testdata.GenerateTraceDataOneEmptyResourceSpans(),
		},
		{
			name:   "one-empty-one-nil-resource-spans",
			input:  testdata.GenerateTraceDataOneEmptyOneNilResourceSpans(),
			output: testdata.GenerateTraceDataOneEmptyOneNilResourceSpans(),
		},
		{
			name:   "no-libraries",
			input:  testdata.GenerateTraceDataNoLibraries(),
			output: testdata.GenerateTraceDataNoLibraries(),
		},
		{
			name:   "one-empty-instrumentation-library",
			input:  testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
			output: testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
		},
		{
			name:   "one-empty-one-nil-instrumentation-library",
			input:  testdata.GenerateTraceDataOneEmptyOneNilInstrumentationLibrary(),
			output: testdata.GenerateTraceDataOneEmptyOneNilInstrumentationLibrary(),
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
		{Key: "attribute1", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tp.ConsumeTrace(context.Background(), tt.input))
			assert.EqualValues(t, tt.output, tt.input)
		})
	}
}

func TestAttributes_InsertValue(t *testing.T) {
	testCases := []testCase{
		// Ensure `attribute1` is set for spans with no attributes.
		{
			name:            "InsertEmptyAttributes",
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		// Ensure `attribute1` is set.
		{
			name: "InsertKeyNoExists",
			inputAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueString("bob"),
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		// Ensures no insert is performed because the keys `attribute1` already exists.
		{
			name: "InsertKeyExists",
			inputAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueString("bob"),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		// Ensure no attribute is inserted because because from_attribute `string_key` does not exist.
		{
			name: "InsertMissingFromAttribute",
			inputAttributes: map[string]data.AttributeValue{
				"bob": data.NewAttributeValueInt(1),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"bob": data.NewAttributeValueInt(1),
			},
		},
		// Ensure `string key` is set.
		{
			name: "InsertAttributeExists",
			inputAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueInt(8892342),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueInt(8892342),
				"string key": data.NewAttributeValueInt(8892342),
			},
		},
		// Ensures no insert is performed because the keys `string key` already exist.
		{
			name: "InsertKeysExists",
			inputAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueInt(8892342),
				"string key": data.NewAttributeValueString("here"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"anotherkey": data.NewAttributeValueInt(8892342),
				"string key": data.NewAttributeValueString("here"),
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "string key", Action: INSERT, FromAttribute: "anotherkey"},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		// Ensure no changes to the span as the key does not exist.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("foo"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("foo"),
			},
		},
		// Ensure the attribute `db.secret` is updated.
		{
			name: "UpdateAttributes",
			inputAttributes: map[string]data.AttributeValue{
				"db.secret": data.NewAttributeValueString("password1234"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"db.secret": data.NewAttributeValueString("redacted"),
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "db.secret", Action: UPDATE, Value: "redacted"},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		// Ensure the attribute `boo` isn't updated because attribute `foo` isn't present in the span.
		{
			name: "UpdateKeyNoExistFromAttribute",
			inputAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("bob"),
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyNoExistMainAttributed",
			inputAttributes: map[string]data.AttributeValue{
				"foo": data.NewAttributeValueString("over there"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo": data.NewAttributeValueString("over there"),
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyFromExistingAttribute",
			inputAttributes: map[string]data.AttributeValue{
				"foo": data.NewAttributeValueString("there is a party over here"),
				"boo": data.NewAttributeValueString("not here"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo": data.NewAttributeValueString("there is a party over here"),
				"boo": data.NewAttributeValueString("there is a party over here"),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "boo", Action: UPDATE, FromAttribute: "foo"},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"region": data.NewAttributeValueString("planet-earth"),
			},
		},
		// Ensure `region` is inserted for spans with some attributes(the key doesn't exist).
		{
			name: "UpsertAttributeNoExist",
			inputAttributes: map[string]data.AttributeValue{
				"mission": data.NewAttributeValueString("to mars"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"mission": data.NewAttributeValueString("to mars"),
				"region":  data.NewAttributeValueString("planet-earth"),
			},
		},
		/// Ensure `region` is updated for spans with the attribute key `region`.
		{
			name: "UpsertAttributeExists",
			inputAttributes: map[string]data.AttributeValue{
				"mission": data.NewAttributeValueString("to mars"),
				"region":  data.NewAttributeValueString("solar system"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"mission": data.NewAttributeValueString("to mars"),
				"region":  data.NewAttributeValueString("planet-earth"),
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "region", Action: UPSERT, Value: "planet-earth"},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			name: "UpsertFromAttributeNoExist",
			inputAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `new_user_key` is inserted for spans with attribute `user_key`.
		{
			name: "UpsertFromAttributeExistsInsert",
			inputAttributes: map[string]data.AttributeValue{
				"user_key": data.NewAttributeValueInt(2245),
				"foo":      data.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user_key":     data.NewAttributeValueInt(2245),
				"new_user_key": data.NewAttributeValueInt(2245),
				"foo":          data.NewAttributeValueString("casper the friendly ghost"),
			},
		},
		// Ensure `new_user_key` is updated for spans with attribute `user_key`.
		{
			name: "UpsertFromAttributeExistsUpdate",
			inputAttributes: map[string]data.AttributeValue{
				"user_key":     data.NewAttributeValueInt(2245),
				"new_user_key": data.NewAttributeValueInt(5422),
				"foo":          data.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user_key":     data.NewAttributeValueInt(2245),
				"new_user_key": data.NewAttributeValueInt(2245),
				"foo":          data.NewAttributeValueString("casper the friendly ghost"),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "new_user_key", Action: UPSERT, FromAttribute: "user_key"},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		// Ensure the span contains no changes because the key doesn't exist.
		{
			name: "DeleteAttributeNoExist",
			inputAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"boo": data.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `duplicate_key` is deleted for spans with the attribute set.
		{
			name: "DeleteAttributeExists",
			inputAttributes: map[string]data.AttributeValue{
				"duplicate_key": data.NewAttributeValueDouble(3245.6),
				"original_key":  data.NewAttributeValueDouble(3245.6),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"original_key": data.NewAttributeValueDouble(3245.6),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "duplicate_key", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	traceData := generateTraceData(
		"",
		"FromAttributeNoChange",
		map[string]data.AttributeValue{
			"boo": data.NewAttributeValueString("ghosts are scary"),
		})

	assert.NoError(t, tp.ConsumeTrace(context.Background(), traceData))
	require.Equal(t, generateTraceData(
		"",
		"FromAttributeNoChange",
		map[string]data.AttributeValue{
			"boo": data.NewAttributeValueString("ghosts are scary"),
		}), traceData)
}

func TestAttributes_Ordering(t *testing.T) {
	testCases := []testCase{
		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. insert `svc.operation`: `default`
		// 3. delete `operation`.
		{
			name: "OrderingApplyAllSteps",
			inputAttributes: map[string]data.AttributeValue{
				"foo": data.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": data.NewAttributeValueString("default"),
			},
		},
		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. insert `svc.operation`: `arithmetic`
		// 3. delete `operation`.
		{
			name: "OrderingOperationExists",
			inputAttributes: map[string]data.AttributeValue{
				"foo":       data.NewAttributeValueString("casper the friendly ghost"),
				"operation": data.NewAttributeValueString("arithmetic"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": data.NewAttributeValueString("arithmetic"),
			},
		},

		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. update `svc.operation` to `default`
		// 3. delete `operation`.
		{
			name: "OrderingSvcOperationExists",
			inputAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": data.NewAttributeValueString("some value"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": data.NewAttributeValueString("default"),
			},
		},

		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. update `svc.operation` to `arithmetic`
		// 3. delete `operation`.
		{
			name: "OrderingBothAttributesExist",
			inputAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"operation":     data.NewAttributeValueString("arithmetic"),
				"svc.operation": data.NewAttributeValueString("add"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"foo":           data.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": data.NewAttributeValueString("arithmetic"),
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

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			serviceName:     "svcB",
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply processor with different value for exclude property",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1":     data.NewAttributeValueInt(123),
				"NoModification": data.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect name for include property",
			serviceName:        "noname",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		{
			name:        "attribute match for exclude property",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}
	oCfg.Include = &filterspan.MatchProperties{
		Services:  []string{"svcA", "svcB.*"},
		MatchType: filterspan.MatchTypeRegexp,
	}
	oCfg.Exclude = &filterspan.MatchProperties{
		Attributes: []filterspan.Attribute{
			{Key: "NoModification", Value: true},
		},
		MatchType: filterspan.MatchTypeStrict,
	}
	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			serviceName:     "svcB",
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1":     data.NewAttributeValueInt(123),
				"NoModification": data.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_span_name",
			serviceName:        "svcB",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		{
			name:               "dont_apply",
			serviceName:        "svcB",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		{
			name:        "incorrect_span_name_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}
	oCfg.Include = &filterspan.MatchProperties{
		SpanNames: []string{"apply", "dont_apply"},
		MatchType: filterspan.MatchTypeStrict,
	}
	oCfg.Exclude = &filterspan.MatchProperties{
		SpanNames: []string{"dont_apply"},
		MatchType: filterspan.MatchTypeStrict,
	}
	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			serviceName:     "svcB",
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply_to_span_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1":     data.NewAttributeValueInt(123),
				"NoModification": data.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_span_name",
			serviceName:        "svcB",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		{
			name:               "apply_dont_apply",
			serviceName:        "svcB",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
		{
			name:        "incorrect_span_name_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(true),
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}
	oCfg.Include = &filterspan.MatchProperties{
		SpanNames: []string{"^apply.*"},
		MatchType: filterspan.MatchTypeRegexp,
	}
	oCfg.Exclude = &filterspan.MatchProperties{
		SpanNames: []string{".*dont_apply$"},
		MatchType: filterspan.MatchTypeRegexp,
	}
	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes: map[string]data.AttributeValue{
				"user.email": data.NewAttributeValueString("john.doe@example.com"),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user.email": data.NewAttributeValueString("73ec53c4ba1747d485ae2a0d7bfafa6cda80a5a9"),
			},
		},
		{
			name: "Int",
			inputAttributes: map[string]data.AttributeValue{
				"user.id": data.NewAttributeValueInt(10),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user.id": data.NewAttributeValueString("71aa908aff1548c8c6cdecf63545261584738a25"),
			},
		},
		{
			name: "Double",
			inputAttributes: map[string]data.AttributeValue{
				"user.balance": data.NewAttributeValueDouble(99.1),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user.balance": data.NewAttributeValueString("76429edab4855b03073f9429fd5d10313c28655e"),
			},
		},
		{
			name: "Bool",
			inputAttributes: map[string]data.AttributeValue{
				"user.authenticated": data.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"user.authenticated": data.NewAttributeValueString("bf8b4530d8d246dd74ac53a13471bba17941dff7"),
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

	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
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
			inputAttributes: map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1": data.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply_to_span_with_attr",
			inputAttributes: map[string]data.AttributeValue{
				"NoModification": data.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]data.AttributeValue{
				"attribute1":     data.NewAttributeValueInt(123),
				"NoModification": data.NewAttributeValueBool(false),
			},
		},
		{
			name:               "dont_apply",
			inputAttributes:    map[string]data.AttributeValue{},
			expectedAttributes: map[string]data.AttributeValue{},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "attribute1", Action: INSERT, Value: 123},
	}
	oCfg.Include = &filterspan.MatchProperties{
		SpanNames: []string{"^apply.*"},
	}
	tp, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	require.Nil(b, err)
	require.NotNil(b, tp)

	for _, tt := range testCases {
		td := generateTraceData(tt.serviceName, tt.name, tt.inputAttributes)

		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				assert.NoError(b, tp.ConsumeTrace(context.Background(), td))
			}
		})

		// Ensure that the modified `td` has the attributes sorted:
		sortAttributes(td)
		require.Equal(b, generateTraceData(tt.serviceName, tt.name, tt.expectedAttributes), td)
	}
}
