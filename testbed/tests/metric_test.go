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

package tests

// This file contains Test functions which initiate the tests. The tests can be either
// coded in this file or use scenarios from perf_scenarios.go.

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/testbed/testbed"
)

func TestMetricNoBackend10kDPSOpenCensus(t *testing.T) {
	tc := testbed.NewTestCase(
		t,
		testbed.NewOCMetricDataSender(55678),
		testbed.NewOCDataReceiver(testbed.DefaultOCPort),
	)
	defer tc.Stop()

	tc.SetResourceLimits(testbed.ResourceSpec{ExpectedMaxCPU: 200, ExpectedMaxRAM: 200})
	tc.StartAgent()

	tc.StartLoad(testbed.LoadOptions{DataItemsPerSecond: 10000})

	tc.Sleep(tc.Duration)
}

func TestMetric10kDPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"OpenCensus",
			testbed.NewOCMetricDataSender(testbed.GetAvailablePort(t)),
			testbed.NewOCDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 60,
			},
		},
		{
			"OTLP",
			testbed.NewOTLPMetricDataSender(testbed.GetAvailablePort(t)),
			testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 60,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				nil,
			)
		})
	}

}

const (
	mockedConsumedResourceWithType          = "mocked_consumed_resource_with_type"
	mockedConsumedResourceWithoutType       = "mocked_consumed_resource_without_type"
	mockedConsumedResourceNil               = "mocked_consumed_resource_nil"
	mockedConsumedResourceWithoutAttributes = "mocked_consumed_resource_without_attributes"
)

func TestMetricResourceProcessor(t *testing.T) {
	inputs, err := ioutil.ReadFile("testdata/resource-processor-inputs.json")
	require.NoError(t, err, "failed to load test dependencies", err)

	mockedConsumedResources := map[string]*otlpmetrics.ResourceMetrics{}
	err = json.Unmarshal(inputs, &mockedConsumedResources)
	require.NoError(t, err, "failed to load test dependencies", err)
	require.Equal(t, 4, len(mockedConsumedResources), "not all mocked resources loaded")

	sender := testbed.NewOTLPMetricDataSender(testbed.GetAvailablePort(t))
	receiver := testbed.NewOTLPDataReceiver(testbed.GetAvailablePort(t))

	tests := []struct {
		name                     string
		resourceProcessorConfig  string
		mockedConsumedMetricData data.MetricData
		expectedMetricData       data.MetricData
		isNilResource            bool
	}{
		{
			name: "Override consumed resource labels and type",
			resourceProcessorConfig: `
  resource:
    type: vm
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					mockedConsumedResources[mockedConsumedResourceWithType],
				},
			),
			expectedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					{
						Resource: &otlpresource.Resource{
							Attributes: []*v1.AttributeKeyValue{
								{
									Key:         "opencensus.resourcetype",
									StringValue: "vm",
								},
								{
									Key:         "label-key",
									StringValue: "label-value",
								},
								{
									Key:         "additional-label-key",
									StringValue: "additional-label-value",
								},
							},
						},
					},
				},
			),
		},
		{
			name: "Return nil if consumed resource is nil and type is empty",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					mockedConsumedResources[mockedConsumedResourceNil],
				},
			),
			isNilResource: true,
		},
		{
			name: "Return nil if consumed resource and resource in config is nil",
			resourceProcessorConfig: `
  resource:
`,
			mockedConsumedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					mockedConsumedResources[mockedConsumedResourceNil],
				},
			),
			isNilResource: true,
		},
		{
			name: "Return resource without type",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					mockedConsumedResources[mockedConsumedResourceWithoutType],
				},
			),
			expectedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					{
						Resource: &otlpresource.Resource{
							Attributes: []*v1.AttributeKeyValue{
								{
									Key:         "label-key",
									StringValue: "label-value",
								},
								{
									Key:         "additional-label-key",
									StringValue: "additional-label-value",
								},
							},
						},
					},
				},
			),
		},
		{
			name: "Consumed resource with nil labels",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					mockedConsumedResources[mockedConsumedResourceWithoutAttributes],
				},
			),
			expectedMetricData: data.MetricDataFromOtlp(
				[]*otlpmetrics.ResourceMetrics{
					{
						Resource: &otlpresource.Resource{
							Attributes: []*v1.AttributeKeyValue{
								{
									Key:         "additional-label-key",
									StringValue: "additional-label-value",
								},
							},
						},
					},
				},
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultDir, err := filepath.Abs(path.Join("results", t.Name()))
			require.NoError(t, err)

			processors := map[string]string{
				"resource": test.resourceProcessorConfig,
			}
			configFile := createConfigFile(t, sender, receiver, resultDir, processors)
			defer os.Remove(configFile)

			require.NotEmpty(t, configFile, "Cannot create config file")

			tc := testbed.NewTestCase(t, sender, receiver, testbed.WithConfigFile(configFile))
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

			// If a resource is not expected to be returned by the processor, return.
			if test.isNilResource {
				require.True(t, rm.At(0).Resource().IsNil())
				return
			}

			require.Equal(t,
				attributesToMap(test.expectedMetricData.ResourceMetrics().At(0).Resource().Attributes()),
				attributesToMap(rm.At(0).Resource().Attributes()),
			)
		})
	}
}
