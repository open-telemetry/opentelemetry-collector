// Copyright 202 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"encoding/json"
	"testing"

	v1 "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/metrics/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/data"
)

const (
	mockedConsumedResourceWithTypeJSON = `
  {
	"resource": {
	  "attributes": [
		{
		  "key": "opencensus.resourcetype",
		  "string_value": "host"
		},
		{
		  "key": "label-key",
		  "string_value": "label-value"
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

	mockedConsumedResourceWithoutTypeJSON = `
  {
    "resource": {
      "attributes": [
        {
          "key": "label-key",
          "string_value": "label-value"
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
	mockedConsumedMetricData data.MetricData
	expectedMetricData       data.MetricData
	isNilResource            bool
}

func getResourceProcessorTestCases(t *testing.T) []resourceProcessorTestCase {

	tests := []resourceProcessorTestCase{
		{
			name: "Override consumed resource labels and type",
			resourceProcessorConfig: `
  resource:
    type: vm
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceWithTypeJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{
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
			}),
		},
		{
			name: "Return nil if consumed resource is nil and type is empty",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceNilJSON),
			isNilResource:            true,
		},
		{
			name: "Return nil if consumed resource and resource in config is nil",
			resourceProcessorConfig: `
  resource:
`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceNilJSON),
			isNilResource:            true,
		},
		{
			name: "Return resource without type",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceWithoutTypeJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{
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
			}),
		},
		{
			name: "Consumed resource with nil labels",
			resourceProcessorConfig: `
  resource:
    labels: {
      "additional-label-key": "additional-label-value",
    }
`,
			mockedConsumedMetricData: getMetricDataFromJSON(t, mockedConsumedResourceWithoutAttributesJSON),
			expectedMetricData: getMetricDataFromResourceMetrics(&otlpmetrics.ResourceMetrics{

				Resource: &otlpresource.Resource{
					Attributes: []*v1.AttributeKeyValue{
						{
							Key:         "additional-label-key",
							StringValue: "additional-label-value",
						},
					},
				},
			}),
		},
	}

	return tests
}

func getMetricDataFromResourceMetrics(rm *otlpmetrics.ResourceMetrics) data.MetricData {
	return data.MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{rm})
}

func getMetricDataFromJSON(t *testing.T, rmString string) data.MetricData {
	var mockedResourceMetrics otlpmetrics.ResourceMetrics

	err := json.Unmarshal([]byte(rmString), &mockedResourceMetrics)
	require.NoError(t, err, "failed to get mocked resource metrics object", err)

	return data.MetricDataFromOtlp([]*otlpmetrics.ResourceMetrics{&mockedResourceMetrics})
}
