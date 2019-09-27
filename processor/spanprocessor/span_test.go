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

package spanprocessor

import (
	"context"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

func TestNewTraceProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	tp, err := NewTraceProcessor(nil, *oCfg)
	require.Error(t, oterr.ErrNilNextConsumer, err)
	require.Nil(t, tp)

	tp, err = NewTraceProcessor(exportertest.NewNopTraceExporter(), *oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
}

// TestSpanProcessor_NilEmpty tests spans and attributes with nil/empty values
// do not cause any errors and no renaming occurs.
func TestSpanProcessor_NilEmpty(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"test-key"}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
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
			{
				Name: &tracepb.TruncatableString{Value: "Empty Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
	assert.Equal(t, consumerdata.TraceData{
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
			{
				Name: &tracepb.TruncatableString{Value: "Empty Attribute Map"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
		},
	}, traceData)
}

// Common structure for the test cases.
type testCase struct {
	inputName        string
	inputAttributes  map[string]*tracepb.AttributeValue
	outputName       string
	outputAttributes map[string]*tracepb.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, tp processor.TraceProcessor) {
	t.Run(tt.inputName, func(t *testing.T) {
		traceData := consumerdata.TraceData{
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.inputName},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.inputAttributes,
					},
				},
			},
		}

		assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))
		require.Equal(t, consumerdata.TraceData{
			Spans: []*tracepb.Span{
				{
					Name: &tracepb.TruncatableString{Value: tt.outputName},
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: tt.outputAttributes,
					},
				},
			},
		}, traceData)
	})
}

// TestSpanProcessor_Values tests all possible value types.
func TestSpanProcessor_Values(t *testing.T) {
	testCases := []testCase{
		{
			inputName: "string type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
			outputName: "bob",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
			},
		},
		{
			inputName: "int type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
			outputName: "123",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
		{
			inputName: "double type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
			},
			outputName: "234.129312",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
			},
		},
		{
			inputName: "bool type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			outputName: "true",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
		},
		{
			inputName: "nil type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": nil,
			},
			outputName: "<nil-attribute-value>",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": nil,
			},
		},
		{
			inputName: "unknown type",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {},
			},
			outputName: "<unknown-attribute-type>",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {},
			},
		},
	}

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
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
			inputName: "first keys missing",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			outputName: "first keys missing",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
		},
		{
			inputName: "middle key missing",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			outputName: "middle key missing",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
		},
		{
			inputName: "last key missing",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
			},
			outputName: "last key missing",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
			},
		},
		{
			inputName: "all keys exists",
			inputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
			outputName: "bob::123::234.129312::true",
			outputAttributes: map[string]*tracepb.AttributeValue{
				"key1": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
				},
				"key2": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"key3": {
					Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
				},
				"key4": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
			},
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2", "key3", "key4"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)
	for _, tc := range testCases {
		runIndividualTestCase(t, tc, tp)
	}
}

// TestSpanProcessor_Separator ensures naming a span with a single key and separator will only contain the value from
// the single key.
func TestSpanProcessor_Separator(t *testing.T) {

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "ensure no separator in the rename with one key"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))

	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "bob"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
					},
				},
			},
		},
	}, traceData)
}

// TestSpanProcessor_NoSeparatorMultipleKeys tests naming a span using multiple keys and no separator.
func TestSpanProcessor_NoSeparatorMultipleKeys(t *testing.T) {

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2"}
	oCfg.Rename.Separator = ""

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "ensure no separator in the rename with two keys"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
						"key2": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))

	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "bob123"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
						"key2": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
					},
				},
			},
		},
	}, traceData)
}

// TestSpanProcessor_SeparatorMultipleKeys tests naming a span with multiple keys and a separator.
func TestSpanProcessor_SeparatorMultipleKeys(t *testing.T) {

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1", "key2", "key3", "key4"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "rename with separators and multiple keys"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
						"key2": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
						"key3": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
						},
						"key4": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))

	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "bob::123::234.129312::true"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
						"key2": {
							Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
						},
						"key3": {
							Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)},
						},
						"key4": {
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
					},
				},
			},
		},
	}, traceData)
}

// TestSpanProcessor_NilName tests naming a span when the input span had no name.
func TestSpanProcessor_NilName(t *testing.T) {

	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Rename.FromAttributes = []string{"key1"}
	oCfg.Rename.Separator = "::"

	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), oCfg)
	require.Nil(t, err)
	require.NotNil(t, tp)

	traceData := consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: nil,
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
					},
				},
			},
		},
	}
	assert.NoError(t, tp.ConsumeTraceData(context.Background(), traceData))

	assert.Equal(t, consumerdata.TraceData{
		Spans: []*tracepb.Span{
			{
				Name: &tracepb.TruncatableString{Value: "bob"},
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key1": {
							Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob"}},
						},
					},
				},
			},
		},
	}, traceData)
}
