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

package attributesprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/translator/conventions"
)

// Common structure for all the Tests
type testCase struct {
	name               string
	serviceName        string
	inputAttributes    map[string]pdata.AttributeValue
	expectedAttributes map[string]pdata.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, tp component.TracesProcessor) {
	t.Run(tt.name, func(t *testing.T) {
		td := generateTraceData(tt.serviceName, tt.name, tt.inputAttributes)
		assert.NoError(t, tp.ConsumeTraces(context.Background(), td))
		// Ensure that the modified `td` has the attributes sorted:
		sortAttributes(td)
		require.Equal(t, generateTraceData(tt.serviceName, tt.name, tt.expectedAttributes), td)
	})
}

func generateTraceData(serviceName, spanName string, attrs map[string]pdata.AttributeValue) pdata.Traces {
	td := pdata.NewTraces()
	td.ResourceSpans().Resize(1)
	rs := td.ResourceSpans().At(0)
	if serviceName != "" {
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

func sortAttributes(td pdata.Traces) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		rs.Resource().Attributes().Sort()
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			spans := ilss.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spans.At(k).Attributes().Sort()
			}
		}
	}
}

// TestSpanProcessor_Values tests all possible value types.
func TestSpanProcessor_NilEmptyData(t *testing.T) {
	type nilEmptyTestCase struct {
		name   string
		input  pdata.Traces
		output pdata.Traces
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
			name:   "no-libraries",
			input:  testdata.GenerateTraceDataNoLibraries(),
			output: testdata.GenerateTraceDataNoLibraries(),
		},
		{
			name:   "one-empty-instrumentation-library",
			input:  testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
			output: testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
		},
	}
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Settings.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
		{Key: "attribute1", Action: processorhelper.DELETE},
	}

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tp.ConsumeTraces(context.Background(), tt.input))
			assert.EqualValues(t, tt.output, tt.input)
		})
	}
}

func TestAttributes_FilterSpans(t *testing.T) {
	testCases := []testCase{
		{
			name:            "apply processor",
			serviceName:     "svcB",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply processor with different value for exclude property",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect name for include property",
			serviceName:        "noname",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:        "attribute match for exclude property",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
	}
	oCfg.Include = &filterconfig.MatchProperties{
		Services: []string{"svcA", "svcB.*"},
		Config:   *createConfig(filterset.Regexp),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		Attributes: []filterconfig.Attribute{
			{Key: "NoModification", Value: true},
		},
		Config: *createConfig(filterset.Strict),
	}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
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
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_span_name",
			serviceName:        "svcB",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:               "dont_apply",
			serviceName:        "svcB",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:        "incorrect_span_name_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
	}
	oCfg.Include = &filterconfig.MatchProperties{
		SpanNames: []string{"apply", "dont_apply"},
		Config:    *createConfig(filterset.Strict),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		SpanNames: []string{"dont_apply"},
		Config:    *createConfig(filterset.Strict),
	}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
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
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name:        "apply_to_span_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_span_name",
			serviceName:        "svcB",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:               "apply_dont_apply",
			serviceName:        "svcB",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:        "incorrect_span_name_with_attr",
			serviceName: "svcB",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(true),
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
	}
	oCfg.Include = &filterconfig.MatchProperties{
		SpanNames: []string{"^apply.*"},
		Config:    *createConfig(filterset.Regexp),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		SpanNames: []string{".*dont_apply$"},
		Config:    *createConfig(filterset.Regexp),
	}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
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
			inputAttributes: map[string]pdata.AttributeValue{
				"user.email": pdata.NewAttributeValueString("john.doe@example.com"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user.email": pdata.NewAttributeValueString("73ec53c4ba1747d485ae2a0d7bfafa6cda80a5a9"),
			},
		},
		{
			name: "Int",
			inputAttributes: map[string]pdata.AttributeValue{
				"user.id": pdata.NewAttributeValueInt(10),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user.id": pdata.NewAttributeValueString("71aa908aff1548c8c6cdecf63545261584738a25"),
			},
		},
		{
			name: "Double",
			inputAttributes: map[string]pdata.AttributeValue{
				"user.balance": pdata.NewAttributeValueDouble(99.1),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user.balance": pdata.NewAttributeValueString("76429edab4855b03073f9429fd5d10313c28655e"),
			},
		},
		{
			name: "Bool",
			inputAttributes: map[string]pdata.AttributeValue{
				"user.authenticated": pdata.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user.authenticated": pdata.NewAttributeValueString("bf8b4530d8d246dd74ac53a13471bba17941dff7"),
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "user.email", Action: processorhelper.HASH},
		{Key: "user.id", Action: processorhelper.HASH},
		{Key: "user.balance", Action: processorhelper.HASH},
		{Key: "user.authenticated", Action: processorhelper.HASH},
	}

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
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
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply_to_span_with_attr",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "dont_apply",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
	}
	oCfg.Include = &filterconfig.MatchProperties{
		SpanNames: []string{"^apply.*"},
	}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
	require.Nil(b, err)
	require.NotNil(b, tp)

	for _, tt := range testCases {
		td := generateTraceData(tt.serviceName, tt.name, tt.inputAttributes)

		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				assert.NoError(b, tp.ConsumeTraces(context.Background(), td))
			}
		})

		// Ensure that the modified `td` has the attributes sorted:
		sortAttributes(td)
		require.Equal(b, generateTraceData(tt.serviceName, tt.name, tt.expectedAttributes), td)
	}
}

func createConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}
