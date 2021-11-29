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

package jsonstream

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestMetricsJSON(t *testing.T) {
	testCases := []struct {
		name    string
		metrics pdata.Metrics
		expect  string
	}{
		{
			"NewMetrics",
			pdata.NewMetrics(),
			``,
		},
		{
			"GenerateMetricsNoLibraries",
			testdata.GenerateMetricsNoLibraries(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceMetric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  }
			}`),
		},
		{
			"GenerateMetricsOneEmptyResourceMetrics",
			testdata.GenerateMetricsOneEmptyResourceMetrics(),
			compactJSON(`{
			  "resource": {
			    "type": "resourceMetric",
			    "labels": {}
			  }
			}`),
		},
		{
			"GenerateMetricsOneEmptyInstrumentationLibrary",
			testdata.GenerateMetricsOneEmptyInstrumentationLibrary(),
			compactJSON(`{
			  "resource": {
			    "type": "instrumentationLibraryMetric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  }
			}`),
		},
		{
			"GeneratMetricsAllTypesWithSampleDatapoints",
			testdata.GeneratMetricsAllTypesWithSampleDatapoints(),
			lines(compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "gauge-int",
			  "description": "",
			  "unit": "1",
			  "dataType": "Gauge",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 123
			    },
			    {
			      "attributes": {
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 456
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "gauge-double",
			  "description": "",
			  "unit": "1",
			  "dataType": "Gauge",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 1.23
			    },
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-3": "label-value-3"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 4.56
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "sum-int",
			  "description": "",
			  "unit": "1",
			  "dataType": "Sum",
			  "isMonotonic": true,
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 123
			    },
			    {
			      "attributes": {
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 456
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "sum-double",
			  "description": "",
			  "unit": "1",
			  "dataType": "Sum",
			  "isMonotonic": true,
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 1.23
			    },
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-3": "label-value-3"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "value": 4.56
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "histogram",
			  "description": "",
			  "unit": "1",
			  "dataType": "Histogram",
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-3": "label-value-3"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "count": 1,
			      "sum": 15,
			      "explicitBounds": [],
			      "bucketCounts": []
			    },
			    {
			      "attributes": {
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "count": 1,
			      "sum": 15,
			      "explicitBounds": [
			        1
			      ],
			      "bucketCounts": [
			        0,
			        1
			      ]
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "exponential-histogram",
			  "description": "",
			  "unit": "1",
			  "dataType": "ExponentialHistogram"
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "summary",
			  "description": "",
			  "unit": "1",
			  "dataType": "Summary",
			  "dataPoints": [
			    {
			      "attributes": {
			        "label-1": "label-value-1",
			        "label-3": "label-value-3"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "count": 1,
			      "sum": 15,
			      "quantileValues": []
			    },
			    {
			      "attributes": {
			        "label-2": "label-value-2"
			      },
			      "startTimestamp": "2020-02-11 20:26:12.000000321 +0000 UTC",
			      "timestamp": "2020-02-11 20:26:13.000000789 +0000 UTC",
			      "count": 1,
			      "sum": 15,
			      "quantileValues": [
			        {
			          "quantile": 0.01,
			          "value": 15
			        }
			      ]
			    }
			  ]
			}`)),
		},
		{
			"GenerateMetricsAllTypesEmptyDataPoint",
			testdata.GenerateMetricsAllTypesEmptyDataPoint(),
			lines(compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "gauge-double",
			  "description": "",
			  "unit": "1",
			  "dataType": "Gauge",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC"
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "gauge-int",
			  "description": "",
			  "unit": "1",
			  "dataType": "Gauge",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC"
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "sum-double",
			  "description": "",
			  "unit": "1",
			  "dataType": "Sum",
			  "isMonotonic": true,
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC"
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "sum-int",
			  "description": "",
			  "unit": "1",
			  "dataType": "Sum",
			  "isMonotonic": true,
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC"
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "histogram",
			  "description": "",
			  "unit": "1",
			  "dataType": "Histogram",
			  "aggregationTemporality": "AGGREGATION_TEMPORALITY_CUMULATIVE",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "count": 0,
			      "sum": 0,
			      "explicitBounds": [],
			      "bucketCounts": []
			    }
			  ]
			}`), compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "summary",
			  "description": "",
			  "unit": "1",
			  "dataType": "Summary",
			  "dataPoints": [
			    {
			      "attributes": {},
			      "startTimestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "timestamp": "1970-01-01 00:00:00 +0000 UTC",
			      "count": 0,
			      "sum": 0,
			      "quantileValues": []
			    }
			  ]
			}`)),
		},
		{
			"GenerateMetricsMetricTypeInvalid",
			testdata.GenerateMetricsMetricTypeInvalid(),
			compactJSON(`{
			  "resource": {
			    "type": "metric",
			    "labels": {
			      "resource-attr": "resource-attr-val-1"
			    }
			  },
			  "instrumentationLibrary": {
			    "name": "",
			    "version": ""
			  },
			  "name": "sum-int",
			  "description": "",
			  "unit": "1",
			  "dataType": "None"
			}`),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := NewJSONMetricsMarshaler().MarshalMetrics(tt.metrics)
			if assert.NoError(t, err) {
				testJSON(t, tt.expect, string(metrics))
			}
		})
	}
}
