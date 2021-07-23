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

package tests

import (
	"context"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/testbed/testbed"
)

var (
	mockedConsumedResourceWithType = func() pdata.Metrics {
		md := pdata.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		rm.Resource().Attributes().UpsertString("opencensus.resourcetype", "host")
		rm.Resource().Attributes().UpsertString("label-key", "label-value")
		m := rm.InstrumentationLibraryMetrics().AppendEmpty().Metrics().AppendEmpty()
		m.SetName("metric-name")
		m.SetDescription("metric-description")
		m.SetUnit("metric-unit")
		m.SetDataType(pdata.MetricDataTypeIntGauge)
		m.IntGauge().DataPoints().AppendEmpty().SetValue(0)
		return md
	}()

	mockedConsumedResourceEmpty = func() pdata.Metrics {
		md := pdata.NewMetrics()
		rm := md.ResourceMetrics().AppendEmpty()
		m := rm.InstrumentationLibraryMetrics().AppendEmpty().Metrics().AppendEmpty()
		m.SetName("metric-name")
		m.SetDescription("metric-description")
		m.SetUnit("metric-unit")
		m.SetDataType(pdata.MetricDataTypeIntGauge)
		m.IntGauge().DataPoints().AppendEmpty().SetValue(0)
		return md
	}()
)

type resourceProcessorTestCase struct {
	name                    string
	resourceProcessorConfig string
	mockedConsumedMetrics   pdata.Metrics
	expectedMetrics         pdata.Metrics
}

func getResourceProcessorTestCases() []resourceProcessorTestCase {

	tests := []resourceProcessorTestCase{
		{
			name: "update_and_rename_existing_attributes",
			resourceProcessorConfig: `
  resource:
    attributes:
    - key: label-key
      value: new-label-value
      action: update
    - key: resource-type
      from_attribute: opencensus.resourcetype
      action: upsert
    - key: opencensus.resourcetype
      action: delete
`,
			mockedConsumedMetrics: mockedConsumedResourceWithType,
			expectedMetrics: func() pdata.Metrics {
				md := pdata.NewMetrics()
				rm := md.ResourceMetrics().AppendEmpty()
				rm.Resource().Attributes().UpsertString("resource-type", "host")
				rm.Resource().Attributes().UpsertString("label-key", "new-label-value")
				return md
			}(),
		},
		{
			name: "set_attribute_on_empty_resource",
			resourceProcessorConfig: `
  resource:
    attributes:
    - key: additional-label-key
      value: additional-label-value
      action: insert

`,
			mockedConsumedMetrics: mockedConsumedResourceEmpty,
			expectedMetrics: func() pdata.Metrics {
				md := pdata.NewMetrics()
				rm := md.ResourceMetrics().AppendEmpty()
				rm.Resource().Attributes().UpsertString("additional-label-key", "additional-label-value")
				return md
			}(),
		},
	}

	return tests
}

func TestMetricResourceProcessor(t *testing.T) {
	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))

	tests := getResourceProcessorTestCases()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			require.NoError(t, err)

			agentProc := testbed.NewChildProcessCollector()
			processors := map[string]string{
				"resource": test.resourceProcessorConfig,
			}
			configStr := createConfigYaml(t, sender, receiver, resultDir, processors, nil)
			configCleanup, err := agentProc.PrepareConfig(configStr)
			require.NoError(t, err)
			defer configCleanup()

			options := testbed.LoadOptions{DataItemsPerSecond: 10000, ItemsPerBatch: 10}
			dataProvider := testbed.NewPerfTestDataProvider(options)
			tc := testbed.NewTestCase(
				t,
				dataProvider,
				sender,
				receiver,
				agentProc,
				&testbed.PerfTestValidator{},
				performanceResultsSummary,
			)
			defer tc.Stop()

			tc.StartBackend()
			tc.StartAgent()
			defer tc.StopAgent()

			tc.EnableRecording()

			require.NoError(t, sender.Start())

			// Clear previously received metrics.
			tc.MockBackend.ClearReceivedItems()
			startCounter := tc.MockBackend.DataItemsReceived()

			sender, ok := tc.Sender.(testbed.MetricDataSender)
			require.True(t, ok, "unsupported metric sender")

			require.NoError(t, sender.ConsumeMetrics(context.Background(), test.mockedConsumedMetrics))

			// We bypass the load generator in this test, but make sure to increment the
			// counter since it is used in final reports.
			tc.LoadGenerator.IncDataItemsSent()

			tc.WaitFor(func() bool { return tc.MockBackend.DataItemsReceived() == startCounter+1 },
				"datapoints received")

			// Assert Resources
			m := tc.MockBackend.ReceivedMetrics[0]
			rm := m.ResourceMetrics()
			require.Equal(t, 1, rm.Len())

			expectidMD := test.expectedMetrics
			require.Equal(t,
				expectidMD.ResourceMetrics().At(0).Resource().Attributes().Sort(),
				rm.At(0).Resource().Attributes().Sort(),
			)
		})
	}
}
