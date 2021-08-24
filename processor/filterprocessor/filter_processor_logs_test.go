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

package filterprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/model/pdata"
)

type logNameTest struct {
	name   string
	exc    []filterconfig.Attribute
	inLogs pdata.Logs
	outLN  [][]string // output Log names per Resource
}

type logWithResource struct {
	logNames           []string
	resourceAttributes map[string]pdata.AttributeValue
}

var (
	inLogNames = []string{
		"full_name_match",
		"random",
	}

	inLogForResourceTest = []logWithResource{
		{
			logNames: []string{"log1", "log2"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val1"),
				"attr2": pdata.NewAttributeValueString("attr2/val2"),
				"attr3": pdata.NewAttributeValueString("attr3/val3"),
			},
		},
	}

	inLogForTwoResource = []logWithResource{
		{
			logNames: []string{"log1", "log2"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val1"),
			},
		},
		{
			logNames: []string{"log3", "log4"},
			resourceAttributes: map[string]pdata.AttributeValue{
				"attr1": pdata.NewAttributeValueString("attr1/val2"),
			},
		},
	}

	standardLogTests = []logNameTest{
		{
			name:   "emptyFilterExclude",
			exc:    []filterconfig.Attribute{},
			inLogs: testResourceLogs([]logWithResource{{logNames: inLogNames}}),
			outLN:  [][]string{inLogNames},
		},
		{
			name:   "excludeNilWithResourceAttributes",
			exc:    []filterconfig.Attribute{},
			inLogs: testResourceLogs(inLogForResourceTest),
			outLN: [][]string{
				{"log1", "log2"},
			},
		},
		{
			name:   "excludeAllWithMissingResourceAttributes",
			exc:    []filterconfig.Attribute{{Key: "attr1", Value: "attr1/val1"}},
			inLogs: testResourceLogs(inLogForTwoResource),
			outLN: [][]string{
				{"log3", "log4"},
			},
		},
	}
)

func TestFilterLogProcessor(t *testing.T) {
	for _, test := range standardLogTests {
		t.Run(test.name, func(t *testing.T) {
			// next stores the results of the filter log processor
			next := new(consumertest.LogsSink)
			cfg := &Config{
				ProcessorSettings: config.NewProcessorSettings(config.NewID(typeStr)),
				Logs: LogFilters{
					ResourceAttributes: test.exc,
				},
			}
			factory := NewFactory()
			flp, err := factory.CreateLogsProcessor(
				context.Background(),
				componenttest.NewNopProcessorCreateSettings(),
				cfg,
				next,
			)
			assert.NotNil(t, flp)
			assert.Nil(t, err)

			caps := flp.Capabilities()
			assert.True(t, caps.MutatesData)
			ctx := context.Background()
			assert.NoError(t, flp.Start(ctx, nil))

			cErr := flp.ConsumeLogs(context.Background(), test.inLogs)
			assert.Nil(t, cErr)
			got := next.AllLogs()

			require.Equal(t, 1, len(got))
			require.Equal(t, len(test.outLN), got[0].ResourceLogs().Len())
			for i, wantOut := range test.outLN {
				gotLogs := got[0].ResourceLogs().At(i).InstrumentationLibraryLogs().At(0).Logs()
				assert.Equal(t, len(wantOut), gotLogs.Len())
				for idx := range wantOut {
					assert.Equal(t, wantOut[idx], gotLogs.At(idx).Name())
				}
			}
			assert.NoError(t, flp.Shutdown(ctx))
		})
	}
}

func testResourceLogs(lwrs []logWithResource) pdata.Logs {
	ld := pdata.NewLogs()

	for _, lwr := range lwrs {
		rl := ld.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().InitFromMap(lwr.resourceAttributes)
		ls := rl.InstrumentationLibraryLogs().AppendEmpty().Logs()
		for _, name := range lwr.logNames {
			l := ls.AppendEmpty()
			l.SetName(name)
		}
	}
	return ld
}

func TestNilResourceLogs(t *testing.T) {
	logs := pdata.NewLogs()
	rls := logs.ResourceLogs()
	rls.AppendEmpty()
	requireNotPanicsLogs(t, logs)
}

func TestNilILL(t *testing.T) {
	logs := pdata.NewLogs()
	rls := logs.ResourceLogs()
	rl := rls.AppendEmpty()
	ills := rl.InstrumentationLibraryLogs()
	ills.AppendEmpty()
	requireNotPanicsLogs(t, logs)
}

func TestNilLog(t *testing.T) {
	logs := pdata.NewLogs()
	rls := logs.ResourceLogs()
	rl := rls.AppendEmpty()
	ills := rl.InstrumentationLibraryLogs()
	ill := ills.AppendEmpty()
	ls := ill.Logs()
	ls.AppendEmpty()
	requireNotPanicsLogs(t, logs)
}

func requireNotPanicsLogs(t *testing.T, logs pdata.Logs) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pcfg := cfg.(*Config)
	pcfg.Logs = LogFilters{
		ResourceAttributes: nil,
	}
	ctx := context.Background()
	proc, _ := factory.CreateLogsProcessor(
		ctx,
		componenttest.NewNopProcessorCreateSettings(),
		cfg,
		consumertest.NewNop(),
	)
	require.NotPanics(t, func() {
		_ = proc.ConsumeLogs(ctx, logs)
	})
}
