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
)

// Common structure for all the Tests
type logTestCase struct {
	name               string
	inputAttributes    map[string]pdata.AttributeValue
	expectedAttributes map[string]pdata.AttributeValue
}

// runIndividualLogTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualLogTestCase(t *testing.T, tt logTestCase, tp component.LogsProcessor) {
	t.Run(tt.name, func(t *testing.T) {
		ld := generateLogData(tt.name, tt.inputAttributes)
		assert.NoError(t, tp.ConsumeLogs(context.Background(), ld))
		// Ensure that the modified `ld` has the attributes sorted:
		sortLogAttributes(ld)
		require.Equal(t, generateLogData(tt.name, tt.expectedAttributes), ld)
	})
}

func generateLogData(logName string, attrs map[string]pdata.AttributeValue) pdata.Logs {
	td := pdata.NewLogs()
	td.ResourceLogs().Resize(1)
	rs := td.ResourceLogs().At(0)
	rs.InstrumentationLibraryLogs().Resize(1)
	ils := rs.InstrumentationLibraryLogs().At(0)
	lrs := ils.Logs()
	lrs.Resize(1)
	lrs.At(0).SetName(logName)
	lrs.At(0).Attributes().InitFromMap(attrs).Sort()
	return td
}

func sortLogAttributes(ld pdata.Logs) {
	rss := ld.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		rs.Resource().Attributes().Sort()
		ilss := rs.InstrumentationLibraryLogs()
		for j := 0; j < ilss.Len(); j++ {
			logs := ilss.At(j).Logs()
			for k := 0; k < logs.Len(); k++ {
				s := logs.At(k)
				s.Attributes().Sort()
			}
		}
	}
}

// TestLogProcessor_Values tests all possible value types.
func TestLogProcessor_NilEmptyData(t *testing.T) {
	type nilEmptyTestCase struct {
		name   string
		input  pdata.Logs
		output pdata.Logs
	}
	testCases := []nilEmptyTestCase{
		{
			name:   "empty",
			input:  testdata.GenerateLogDataEmpty(),
			output: testdata.GenerateLogDataEmpty(),
		},
		{
			name:   "one-empty-resource-logs",
			input:  testdata.GenerateLogDataOneEmptyResourceLogs(),
			output: testdata.GenerateLogDataOneEmptyResourceLogs(),
		},
		{
			name:   "no-libraries",
			input:  testdata.GenerateLogDataOneEmptyResourceLogs(),
			output: testdata.GenerateLogDataOneEmptyResourceLogs(),
		},
	}
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Settings.Actions = []processorhelper.ActionKeyValue{
		{Key: "attribute1", Action: processorhelper.INSERT, Value: 123},
		{Key: "attribute1", Action: processorhelper.DELETE},
	}

	tp, err := factory.CreateLogsProcessor(
		context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewLogsNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
	for i := range testCases {
		tt := testCases[i]
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tp.ConsumeLogs(context.Background(), tt.input))
			assert.EqualValues(t, tt.output, tt.input)
		})
	}
}

func TestAttributes_FilterLogs(t *testing.T) {
	testCases := []logTestCase{
		{
			name:            "apply processor",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply processor with different value for exclude property",
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
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name: "attribute match for exclude property",
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
		LogNames: []string{"^[^i].*"},
		Config:   *createConfig(filterset.Regexp),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		Attributes: []filterconfig.Attribute{
			{Key: "NoModification", Value: true},
		},
		Config: *createConfig(filterset.Strict),
	}
	tp, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestAttributes_FilterLogsByNameStrict(t *testing.T) {
	testCases := []logTestCase{
		{
			name:            "apply",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_log_name",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:               "dont_apply",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name: "incorrect_log_name_with_attr",
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
		LogNames: []string{"apply", "dont_apply"},
		Config:   *createConfig(filterset.Strict),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		LogNames: []string{"dont_apply"},
		Config:   *createConfig(filterset.Strict),
	}
	tp, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestAttributes_FilterLogsByNameRegexp(t *testing.T) {
	testCases := []logTestCase{
		{
			name:            "apply_to_log_with_no_attrs",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply_to_log_with_attr",
			inputAttributes: map[string]pdata.AttributeValue{
				"NoModification": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1":     pdata.NewAttributeValueInt(123),
				"NoModification": pdata.NewAttributeValueBool(false),
			},
		},
		{
			name:               "incorrect_log_name",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name:               "apply_dont_apply",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		{
			name: "incorrect_log_name_with_attr",
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
		LogNames: []string{"^apply.*"},
		Config:   *createConfig(filterset.Regexp),
	}
	oCfg.Exclude = &filterconfig.MatchProperties{
		LogNames: []string{".*dont_apply$"},
		Config:   *createConfig(filterset.Regexp),
	}
	tp, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestLogAttributes_Hash(t *testing.T) {
	testCases := []logTestCase{
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

	tp, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.Nil(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func BenchmarkAttributes_FilterLogsByName(b *testing.B) {
	testCases := []logTestCase{
		{
			name:            "apply_to_log_with_no_attrs",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		{
			name: "apply_to_log_with_attr",
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
		LogNames: []string{"^apply.*"},
	}
	tp, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.Nil(b, err)
	require.NotNil(b, tp)

	for _, tt := range testCases {
		td := generateLogData(tt.name, tt.inputAttributes)

		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				assert.NoError(b, tp.ConsumeLogs(context.Background(), td))
			}
		})

		// Ensure that the modified `td` has the attributes sorted:
		sortLogAttributes(td)
		require.Equal(b, generateLogData(tt.name, tt.expectedAttributes), td)
	}
}
