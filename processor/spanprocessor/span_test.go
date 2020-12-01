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

package spanprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestNewTraceProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"foo"}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, nil)
	require.Error(t, componenterror.ErrNilNextConsumer, err)
	require.Nil(t, tp)

	tp, err = factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
}

// Common structure for the test cases.
type testCase struct {
	serviceName      string
	inputName        string
	inputAttributes  map[string]pdata.AttributeValue
	outputName       string
	outputAttributes map[string]pdata.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, tp component.TracesProcessor) {
	t.Run(tt.inputName, func(t *testing.T) {
		td := generateTraceData(tt.serviceName, tt.inputName, tt.inputAttributes)

		assert.NoError(t, tp.ConsumeTraces(context.Background(), td))
		// Ensure that the modified `td` has the attributes sorted:
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
		assert.EqualValues(t, generateTraceData(tt.serviceName, tt.outputName, tt.outputAttributes), td)
	})
}

func generateTraceData(serviceName, inputName string, attrs map[string]pdata.AttributeValue) pdata.Traces {
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
	spans.At(0).SetName(inputName)
	spans.At(0).Attributes().InitFromMap(attrs).Sort()
	return td
}

// TestSpanProcessor_Values tests all possible value types.
func TestSpanProcessor_NilEmptyData(t *testing.T) {
	type nilEmptyTestCase struct {
		name   string
		input  pdata.Traces
		output pdata.Traces
	}
	// TODO: Add test for "nil" Span. This needs support from data slices to allow to construct that.
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
	oCfg.Include = &filterconfig.MatchProperties{
		Config:   *createMatchConfig(filterset.Strict),
		Services: []string{"service"},
	}
	oCfg.Rename.FromAttributes = []string{"key"}

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

// TestSpanProcessor_Values tests all possible value types.
func TestSpanProcessor_Values(t *testing.T) {
	// TODO: Add test for "nil" Span. This needs support from data slices to allow to construct that.
	testCases := []testCase{
		{
			inputName:        "",
			inputAttributes:  nil,
			outputName:       "",
			outputAttributes: nil,
		},
		{
			inputName:        "nil-attributes",
			inputAttributes:  nil,
			outputName:       "nil-attributes",
			outputAttributes: nil,
		},
		{
			inputName:        "empty-attributes",
			inputAttributes:  map[string]pdata.AttributeValue{},
			outputName:       "empty-attributes",
			outputAttributes: map[string]pdata.AttributeValue{},
		},
		{
			inputName: "string-type",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
			},
			outputName: "bob",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
			},
		},
		{
			inputName: "int-type",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueInt(123),
			},
			outputName: "123",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			inputName: "double-type",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueDouble(234.129312),
			},
			outputName: "234.129312",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueDouble(234.129312),
			},
		},
		{
			inputName: "bool-type",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueBool(true),
			},
			outputName: "true",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueBool(true),
			},
		},
		// TODO: What do we do when AttributeMap contains a nil entry? Is that possible?
		// TODO: In the new protocol do we want to support unknown type as 0 instead of string?
		// TODO: Do we want to allow constructing entries with unknown type?
		/*{
			inputName: "nil-type",
			inputAttributes: map[string]data.AttributeValue{
				"key1": data.NewAttributeValue(),
			},
			outputName: "<nil-attribute-value>",
			outputAttributes: map[string]data.AttributeValue{
				"key1": data.NewAttributeValue(),
			},
		},
		{
			inputName: "unknown-type",
			inputAttributes: map[string]data.AttributeValue{
				"key1": {},
			},
			outputName: "<unknown-attribute-type>",
			outputAttributes: map[string]data.AttributeValue{
				"key1": {},
			},
		},*/
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
	for _, tc := range testCases {
		runIndividualTestCase(t, tc, tp)
	}
}

// TestSpanProcessor_MissingKeys tests that missing a key in an attribute map results in no span name changes.
func TestSpanProcessor_MissingKeys(t *testing.T) {
	testCases := []testCase{
		{
			inputName: "first-keys-missing",
			inputAttributes: map[string]pdata.AttributeValue{
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
				"key4": pdata.NewAttributeValueBool(true),
			},
			outputName: "first-keys-missing",
			outputAttributes: map[string]pdata.AttributeValue{
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
				"key4": pdata.NewAttributeValueBool(true),
			},
		},
		{
			inputName: "middle-key-missing",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key4": pdata.NewAttributeValueBool(true),
			},
			outputName: "middle-key-missing",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key4": pdata.NewAttributeValueBool(true),
			},
		},
		{
			inputName: "last-key-missing",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
			},
			outputName: "last-key-missing",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
			},
		},
		{
			inputName: "all-keys-exists",
			inputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
				"key4": pdata.NewAttributeValueBool(true),
			},
			outputName: "bob::123::234.129312::true",
			outputAttributes: map[string]pdata.AttributeValue{
				"key1": pdata.NewAttributeValueString("bob"),
				"key2": pdata.NewAttributeValueInt(123),
				"key3": pdata.NewAttributeValueDouble(234.129312),
				"key4": pdata.NewAttributeValueBool(true),
			},
		},
	}
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2", "key3", "key4"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
	for _, tc := range testCases {
		runIndividualTestCase(t, tc, tp)
	}
}

// TestSpanProcessor_Separator ensures naming a span with a single key and separator will only contain the value from
// the single key.
func TestSpanProcessor_Separator(t *testing.T) {

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := generateTraceData(
		"",
		"ensure no separator in the rename with one key",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
		})
	assert.NoError(t, tp.ConsumeTraces(context.Background(), traceData))

	assert.Equal(t, generateTraceData(
		"",
		"bob",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
		}), traceData)
}

// TestSpanProcessor_NoSeparatorMultipleKeys tests naming a span using multiple keys and no separator.
func TestSpanProcessor_NoSeparatorMultipleKeys(t *testing.T) {

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2"}
	oCfg.Rename.Separator = ""

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := generateTraceData(
		"",
		"ensure no separator in the rename with two keys", map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
			"key2": pdata.NewAttributeValueInt(123),
		})
	assert.NoError(t, tp.ConsumeTraces(context.Background(), traceData))

	assert.Equal(t, generateTraceData(
		"",
		"bob123",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
			"key2": pdata.NewAttributeValueInt(123),
		}), traceData)
}

// TestSpanProcessor_SeparatorMultipleKeys tests naming a span with multiple keys and a separator.
func TestSpanProcessor_SeparatorMultipleKeys(t *testing.T) {

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2", "key3", "key4"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := generateTraceData(
		"",
		"rename with separators and multiple keys",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
			"key2": pdata.NewAttributeValueInt(123),
			"key3": pdata.NewAttributeValueDouble(234.129312),
			"key4": pdata.NewAttributeValueBool(true),
		})
	assert.NoError(t, tp.ConsumeTraces(context.Background(), traceData))

	assert.Equal(t, generateTraceData(
		"",
		"bob::123::234.129312::true",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
			"key2": pdata.NewAttributeValueInt(123),
			"key3": pdata.NewAttributeValueDouble(234.129312),
			"key4": pdata.NewAttributeValueBool(true),
		}), traceData)
}

// TestSpanProcessor_NilName tests naming a span when the input span had no name.
func TestSpanProcessor_NilName(t *testing.T) {

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := generateTraceData(
		"",
		"",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
		})
	assert.NoError(t, tp.ConsumeTraces(context.Background(), traceData))

	assert.Equal(t, generateTraceData(
		"",
		"bob",
		map[string]pdata.AttributeValue{
			"key1": pdata.NewAttributeValueString("bob"),
		}), traceData)
}

// TestSpanProcessor_ToAttributes
func TestSpanProcessor_ToAttributes(t *testing.T) {

	testCases := []struct {
		rules           []string
		breakAfterMatch bool
		testCase
	}{
		{
			rules: []string{`^\/api\/v1\/document\/(?P<documentId>.*)\/update\/1$`},
			testCase: testCase{
				inputName:       "/api/v1/document/321083210/update/1",
				inputAttributes: map[string]pdata.AttributeValue{},
				outputName:      "/api/v1/document/{documentId}/update/1",
				outputAttributes: map[string]pdata.AttributeValue{
					"documentId": pdata.NewAttributeValueString("321083210"),
				},
			},
		},

		{
			rules: []string{`^\/api\/(?P<version>.*)\/document\/(?P<documentId>.*)\/update\/2$`},
			testCase: testCase{
				inputName:  "/api/v1/document/321083210/update/2",
				outputName: "/api/{version}/document/{documentId}/update/2",
				outputAttributes: map[string]pdata.AttributeValue{
					"documentId": pdata.NewAttributeValueString("321083210"),
					"version":    pdata.NewAttributeValueString("v1"),
				},
			},
		},

		{
			rules: []string{`^\/api\/.*\/document\/(?P<documentId>.*)\/update\/3$`,
				`^\/api\/(?P<version>.*)\/document\/.*\/update\/3$`},
			testCase: testCase{
				inputName:  "/api/v1/document/321083210/update/3",
				outputName: "/api/{version}/document/{documentId}/update/3",
				outputAttributes: map[string]pdata.AttributeValue{
					"documentId": pdata.NewAttributeValueString("321083210"),
					"version":    pdata.NewAttributeValueString("v1"),
				},
			},
			breakAfterMatch: false,
		},

		{
			rules: []string{`^\/api\/v1\/document\/(?P<documentId>.*)\/update\/4$`,
				`^\/api\/(?P<version>.*)\/document\/(?P<documentId>.*)\/update\/4$`},
			testCase: testCase{
				inputName:  "/api/v1/document/321083210/update/4",
				outputName: "/api/v1/document/{documentId}/update/4",
				outputAttributes: map[string]pdata.AttributeValue{
					"documentId": pdata.NewAttributeValueString("321083210"),
				},
			},
			breakAfterMatch: true,
		},

		{
			rules: []string{"rule"},
			testCase: testCase{
				inputName:        "",
				outputName:       "",
				outputAttributes: nil,
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.ToAttributes = &ToAttributes{}

	for _, tc := range testCases {
		oCfg.Rename.ToAttributes.Rules = tc.rules
		oCfg.Rename.ToAttributes.BreakAfterMatch = tc.breakAfterMatch
		tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
		require.Nil(t, err)
		require.NotNil(t, tp)

		runIndividualTestCase(t, tc.testCase, tp)
	}
}

func TestSpanProcessor_skipSpan(t *testing.T) {
	testCases := []testCase{
		{
			serviceName: "bankss",
			inputName:   "url/url",
			outputName:  "url/url",
		},
		{
			serviceName: "banks",
			inputName:   "noslasheshere",
			outputName:  "noslasheshere",
		},
		{
			serviceName: "banks",
			inputName:   "www.test.com/code",
			outputName:  "{operation_website}",
			outputAttributes: map[string]pdata.AttributeValue{
				"operation_website": pdata.NewAttributeValueString("www.test.com/code"),
			},
		},
		{
			serviceName: "banks",
			inputName:   "donot/",
			inputAttributes: map[string]pdata.AttributeValue{
				"operation_website": pdata.NewAttributeValueString("www.test.com/code"),
			},
			outputName: "{operation_website}",
			outputAttributes: map[string]pdata.AttributeValue{
				"operation_website": pdata.NewAttributeValueString("donot/"),
			},
		},
		{
			serviceName: "banks",
			inputName:   "donot/change",
			inputAttributes: map[string]pdata.AttributeValue{
				"operation_website": pdata.NewAttributeValueString("www.test.com/code"),
			},
			outputName: "donot/change",
			outputAttributes: map[string]pdata.AttributeValue{
				"operation_website": pdata.NewAttributeValueString("www.test.com/code"),
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Include = &filterconfig.MatchProperties{
		Config:    *createMatchConfig(filterset.Regexp),
		Services:  []string{`^banks$`},
		SpanNames: []string{"/"},
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		Config:    *createMatchConfig(filterset.Strict),
		SpanNames: []string{`donot/change`},
	}
	oCfg.Rename.ToAttributes = &ToAttributes{
		Rules: []string{`(?P<operation_website>.*?)$`},
	}
	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tc := range testCases {
		runIndividualTestCase(t, tc, tp)
	}
}
