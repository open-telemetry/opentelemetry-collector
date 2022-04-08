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

package pmetric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var metricsOTLP = func() Metrics {
	md := NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().UpsertString("host.name", "testHost")
	il := rm.ScopeMetrics().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Metrics().AppendEmpty().SetName("testMetric")
	return md
}()

var metricsJSON = `{"resourceMetrics":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}]},"scopeMetrics":[{"scope":{"name":"name","version":"version"},"metrics":[{"name":"testMetric"}]}]}]}`

func TestMetricsJSON(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)

	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err = decoder.UnmarshalMetrics(jsonBuf)
	assert.NoError(t, err)

	assert.EqualValues(t, metricsOTLP, got)
}

func TestMetricsJSON_Marshal(t *testing.T) {
	encoder := NewJSONMarshaler()
	jsonBuf, err := encoder.MarshalMetrics(metricsOTLP)
	assert.NoError(t, err)
	assert.Equal(t, metricsJSON, string(jsonBuf))
}

func TestMetricsNil(t *testing.T) {
	jsonBuf := `{
"resourceMetrics": [
	{
	"resource": {
		"attributes": [
		{
			"key": "service.name",
			"value": {
			"stringValue": "unknown_service:node"
			}
		},
		{
			"key": "telemetry.sdk.language",
			"value": {
			"stringValue": "nodejs"
			}
		},
		{
			"key": "telemetry.sdk.name",
			"value": {
			"stringValue": "opentelemetry"
			}
		},
		{
			"key": "telemetry.sdk.version",
			"value": {
			"stringValue": "0.24.0"
			}
		}
		],
		"droppedAttributesCount": 0
	},
	"instrumentationLibraryMetrics": [
		{
		"metrics": [
			{
			"name": "metric_name",
			"description": "Example of a UpDownCounter",
			"unit": "1",
			"doubleSum": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "pid",
						"value": "50712"
					}
					],
					"value": 1,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056185378763800
				}
				],
				"isMonotonic": false,
				"aggregationTemporality": 2
			}
			},
			{
			"name": "your_metric_name",
			"description": "Example of a sync observer with callback",
			"unit": "1",
			"doubleGauge": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "label",
						"value": "1"
					}
					],
					"value": 0.07604853280317792,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056189394600700
				}
				]
			}
			},
			{
			"name": "your_metric_name",
			"description": "Example of a sync observer with callback",
			"unit": "1",
			"doubleGauge": {
				"dataPoints": [
				{
					"labels": [
					{
						"key": "label",
						"value": "2"
					}
					],
					"value": 0.9332005145656965,
					"startTimeUnixNano": 1631056185376000000,
					"timeUnixNano": 1631056189394630400
				}
				]
			}
			}
		],
		"instrumentationLibrary": {
			"name": "example-meter"
		}
		}
	]
	}
]
}`
	decoder := NewJSONUnmarshaler()
	var got interface{}
	got, err := decoder.UnmarshalMetrics([]byte(jsonBuf))
	assert.Error(t, err)

	assert.EqualValues(t, Metrics{}, got)
}
