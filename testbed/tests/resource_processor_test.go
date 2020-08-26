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
	"path"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/resource/v1"
	"go.opentelemetry.io/collector/testbed/testbed"
)

const (
	mockedConsumedResourceWithTypeJSON = `
  {
	"resource": {
	  "attributes": [
		{
		  "key": "opencensus.resourcetype",
		  "value": { "stringValue": "host" }
		},
		{
		  "key": "label-key",
		  "value": { "stringValue": "label-value" }
		}
	  ]
	},
	"instrumentation_library_metrics": [
	  {
		"metrics": [
		  {
			"metric_descriptor": {
			  "name": "metric-name",
			  "description": "metric-description",
			  "unit": "metric-unit",
			  "type": 1
			},
			"int64_data_points": [
			  {
				"value": 0
			  }
			]
		  }
		]
	  }
	]
  }
`

	mockedConsumedResourceNilJSON = `
  {
    "instrumentation_library_metrics": [
      {
        "metrics": [
          {
            "metric_descriptor": {
              "name": "metric-name",
              "description": "metric-description",
              "unit": "metric-unit",
              "type": 1
            },
            "int64_data_points": [
              {
                "value": 0
              }
            ]
          }
        ]
      }
    ]
  }
`
	mockedConsumedResourceWithoutAttributesJSON = `
  {
    "resource": {},
    "instrumentation_library_metrics": [
      {
        "metrics": [
          {
            "metric_descriptor": {
              "name": "metric-name",
              "description": "metric-description",
              "unit": "metric-unit",
              "type": 1
            },
            "int64_data_points": [
              {
                "value": 0
              }
            ]
          }
        ]
      }
    ]
  }
`
)

type resourceProcessorTestCase struct {
	name                     string
	resourceProcessorConfig  string
	mockedConsumedMetricData pdata.Metrics
	expectedMetricData       pdata.Metrics
}

func getResourceProcessorTestCases(t *testing.T) []resourceProcessorTestCase {

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
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceWithTypeJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{
				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   "resource-type",
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "host"}},
						},
						{
							Key:   "label-key",
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "new-label-value"}},
						},
					},
				},
			}),
		},
		{
			name: "set_attribute_on_nil_resource",
			resourceProcessorConfig: `
  resource:
    attributes:
    - key: additional-label-key
      value: additional-label-value
      action: insert

`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceNilJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{

				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   "additional-label-key",
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "additional-label-value"}},
						},
					},
				},
			}),
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
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceWithoutAttributesJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{

				Resource: &otlpresource.Resource{
					Attributes: []*otlpcommon.KeyValue{
						{
							Key:   "additional-label-key",
							Value: &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "additional-label-value"}},
						},
					},
				},
			}),
		},
	}

	return tests
}

func getMetricDataFromResourceMetrics(rm *otlpmetrics.ResourceMetrics) pdata.Metrics {
	return pdatautil.MetricsFromInternalMetrics(data.MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{rm}))
}

func getMetricDataFromJSON(t *testing.T, rmString string) pdata.Metrics {
	var mockedResourceMetrics otlpmetrics.ResourceMetrics

	err := jsonpb.UnmarshalString(rmString, &mockedResourceMetrics)
	require.NoError(t, err, "failed to get mocked resource metrics object", err)

	return pdatautil.MetricsFromInternalMetrics(data.MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{&mockedResourceMetrics}))
}

func TestMetricResourceProcessor(t *testing.T) {
	sender := testbed.NewOTLPMetricDataSender(testbed.DefaultHost, testbed.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))

	tests := getResourceProcessorTestCases(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			require.NoError(t, err)

			agentProc := &testbed.ChildProcess{}
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

			sender.Start()

			// Clear previously received metrics.
			tc.MockBackend.ClearReceivedItems()
			startCounter := tc.MockBackend.DataItemsReceived()

			sender, ok := tc.Sender.(testbed.MetricDataSender)
			require.True(t, ok, "unsupported metric sender")

			err = sender.SendMetrics(test.mockedConsumedMetricData)
			require.NoError(t, err, "failed to send metrics", err)

			// We bypass the load generator in this test, but make sure to increment the
			// counter since it is used in final reports.
			tc.LoadGenerator.IncDataItemsSent()

			tc.WaitFor(func() bool { return tc.MockBackend.DataItemsReceived() == startCounter+1 },
				"datapoints received")

			// Assert Resources
			m := tc.MockBackend.ReceivedMetrics[0]
			rm := pdatautil.MetricsToInternalMetrics(m).ResourceMetrics()
			require.Equal(t, 1, rm.Len())

			expectidMD := pdatautil.MetricsToInternalMetrics(test.expectedMetricData)
			require.Equal(t,
				attributesToMap(expectidMD.ResourceMetrics().At(0).Resource().Attributes()),
				attributesToMap(rm.At(0).Resource().Attributes()),
			)
		})
	}
}
