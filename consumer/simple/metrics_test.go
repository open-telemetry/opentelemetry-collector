// Copyright 2020 The OpenTelemetry Authors
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

package simple

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testutil/metricstestutil"
)

func ExampleMetrics() {
	metrics := pdata.NewMetrics()

	mb := Metrics{
		Metrics:                       metrics,
		InstrumentationLibraryName:    "example",
		InstrumentationLibraryVersion: "0.1",
		ResourceAttributes: map[string]string{
			"host": "my-host",
		},
		Timestamp: time.Now(),
	}

	for _, disk := range []string{"sda", "sdb", "sdc"} {
		// All metrics added after this will have these labels
		diskBuilder := mb.WithLabels(map[string]string{
			"disk": disk,
		}).AddGaugeDataPoint("disk.usage", 9000000)

		// Add metrics in a chained manner
		diskBuilder.
			AddGaugeDataPoint("disk.capacity", 9000000).
			AddDGaugeDataPoint("disk.temp", 30.5)

		// Add additional labels
		diskBuilder.WithLabels(map[string]string{
			"direction": "read",
		}).AddSumDataPoint("disk.ops", 50)

		diskBuilder.WithLabels(map[string]string{
			"direction": "write",
		}).AddSumDataPoint("disk.ops", 80)
	}

	metricCount, dpCount := metrics.MetricAndDataPointCount()
	fmt.Printf("Metrics: %d\nDataPoints: %d", metricCount, dpCount)

	// Do not reuse Metrics once you are done using the generated Metrics. Make
	// a new instance of it along with a new instance of pdata.Metrics.

	// Output:
	// Metrics: 4
	// DataPoints: 15
}

func TestMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()

	mb := Metrics{
		Metrics:                       metrics,
		InstrumentationLibraryName:    "example",
		InstrumentationLibraryVersion: "0.1",
		ResourceAttributes: map[string]string{
			"host": "my-host",
		},
		Timestamp: time.Unix(0, 1597266546570840817),
		Labels: map[string]string{
			"disk": "sda",
		},
	}

	expected := `[
  {
    "resource": {
      "attributes": [
        {
          "key": "host",
          "value": {
            "Value": {
              "string_value": "my-host"
            }
          }
        }
      ]
    },
    "instrumentation_library_metrics": [
      {
        "instrumentation_library": {
          "name": "example",
          "version": "0.1"
        },
        "metrics": [
          {
            "name": "disk.capacity",
            "Data": {
              "int_gauge": {
                "data_points": [
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 9000000,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "disk.reads",
            "Data": {
              "int_sum": {
                "data_points": [
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 50,
                    "exemplars": null
                  },
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      },
                      {
                        "key": "partition",
                        "value": "1"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 5,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "disk.temp",
            "Data": {
              "double_gauge": {
                "data_points": [
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 30.5,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "disk.time_awake",
            "Data": {
              "double_sum": {
                "data_points": [
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 100.6,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "partition.capacity",
            "Data": {
              "int_gauge": {
                "data_points": [
                  {
                    "labels": [
                      {
                        "key": "disk",
                        "value": "sda"
                      },
                      {
                        "key": "partition",
                        "value": "1"
                      }
                    ],
                    "time_unix_nano": 1597266546570840817,
                    "value": 40000,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "disk.times",
            "Data": {
              "int_histogram": {
                "data_points": [
                  {
                    "labels": [],
                    "time_unix_nano": 1597266546570840817,
                    "exemplars": null
                  }
                ]
              }
            }
          },
          {
            "name": "disk.double_times",
            "Data": {
              "double_histogram": {
                "data_points": [
                  {
                    "labels": [],
                    "time_unix_nano": 1597266546570840817,
                    "exemplars": null
                  }
                ]
              }
            }
          }
        ]
      }
    ]
  }
]`

	mb.
		AddGaugeDataPoint("disk.capacity", 9000000).
		AddSumDataPoint("disk.reads", 50).
		AddDGaugeDataPoint("disk.temp", 30.5).
		AddDSumDataPoint("disk.time_awake", 100.6)

	intHisto := pdata.NewIntHistogramDataPoint()
	doubleHisto := pdata.NewDoubleHistogramDataPoint()

	mb.WithLabels(map[string]string{
		"partition": "1",
	}).
		AddGaugeDataPoint("partition.capacity", 40000).
		AddSumDataPoint("disk.reads", 5).
		AddHistogramRawDataPoint("disk.times", intHisto).
		AddDHistogramRawDataPoint("disk.double_times", doubleHisto)

	mCount, dpCount := metrics.MetricAndDataPointCount()
	require.Equal(t, 7, mCount)
	require.Equal(t, 8, dpCount)
	asJSON, _ := json.MarshalIndent(pdata.MetricsToOtlp(metricstestutil.SortedMetrics(metrics)), "", "  ")
	require.Equal(t, expected, string(asJSON))
}

func TestMetricFactories(t *testing.T) {
	mb := Metrics{
		Metrics: pdata.NewMetrics(),
		MetricFactoriesByName: map[string]func() pdata.Metric{
			"disk.ops": func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetName("disk.ops")
				m.SetDescription("This counts disk operations")
				m.SetDataType(pdata.MetricDataTypeIntSum)
				return m
			},
		},
		InstrumentationLibraryName:    "example",
		InstrumentationLibraryVersion: "0.1",
		ResourceAttributes: map[string]string{
			"host": "my-host",
		},
		Timestamp: time.Unix(0, 1597266546570840817),
		Labels: map[string]string{
			"disk": "sda",
		},
	}

	mb.WithLabels(map[string]string{
		"direction": "read",
	}).AddSumDataPoint("disk.ops", 5)

	mb.WithLabels(map[string]string{
		"direction": "write",
	}).AddSumDataPoint("disk.ops", 5)

	rms := mb.Metrics.ResourceMetrics()
	require.Equal(t, 1, rms.Len())

	ilms := rms.At(0).InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	require.Equal(t, 1, ilms.At(0).Metrics().Len())
	m := ilms.At(0).Metrics().At(0)
	require.Equal(t, "disk.ops", m.Name())
	require.Equal(t, "This counts disk operations", m.Description())
	require.Equal(t, pdata.MetricDataTypeIntSum, m.DataType())
	require.Equal(t, 2, m.IntSum().DataPoints().Len())

	require.PanicsWithError(t, `mismatched metric data types for metric "disk.ops": "IntSum" vs "IntGauge"`, func() {
		mb.AddGaugeDataPoint("disk.ops", 1)
	})

	mb.AddGaugeDataPoint("disk.temp", 25)
	require.Equal(t, 2, ilms.At(0).Metrics().Len())
	m = ilms.At(0).Metrics().At(1)
	require.Equal(t, "disk.temp", m.Name())
	require.Equal(t, "", m.Description())
	require.Equal(t, pdata.MetricDataTypeIntGauge, m.DataType())
	require.Equal(t, 1, m.IntGauge().DataPoints().Len())
}

func ExampleSafeMetrics() {
	metrics := pdata.NewMetrics()

	mb := Metrics{
		Metrics:                       metrics,
		InstrumentationLibraryName:    "example",
		InstrumentationLibraryVersion: "0.1",
		ResourceAttributes: map[string]string{
			"host": "my-host",
		},
		Timestamp: time.Now(),
	}.AsSafe()

	var wg sync.WaitGroup
	for _, disk := range []string{"sda", "sdb", "sdc"} {
		wg.Add(1)
		go func(disk string) {
			// All metrics added after this will have these labels
			diskBuilder := mb.WithLabels(map[string]string{
				"disk": disk,
			}).AddGaugeDataPoint("disk.usage", 9000000)

			// Add metrics in a chained manner
			diskBuilder.
				AddGaugeDataPoint("disk.capacity", 9000000).
				AddSumDataPoint("disk.reads", 50)

			// Or add them on their own
			diskBuilder.AddDGaugeDataPoint("disk.temp", 30.5)

			wg.Done()
		}(disk)
	}
	wg.Wait()

	metricCount, dpCount := metrics.MetricAndDataPointCount()
	fmt.Printf("Metrics: %d\nDataPoints: %d", metricCount, dpCount)

	// Output:
	// Metrics: 4
	// DataPoints: 12
}

func TestSafeMetrics(t *testing.T) {
	metrics := pdata.NewMetrics()

	mb := Metrics{
		Metrics:                       metrics,
		InstrumentationLibraryName:    "example",
		InstrumentationLibraryVersion: "0.1",
		ResourceAttributes: map[string]string{
			"host": "my-host",
		},
		Timestamp: time.Unix(0, 1597266546570840817),
		Labels: map[string]string{
			"disk": "sda",
		},
	}.AsSafe()

	ready := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(idx string) {
			<-ready
			mb.
				AddGaugeDataPoint("disk.capacity"+idx, 9000000).
				AddSumDataPoint("disk.reads"+idx, 50).
				AddDGaugeDataPoint("disk.temp"+idx, 30.5).
				AddDSumDataPoint("disk.time_awake"+idx, 100.6)

			intHisto := pdata.NewIntHistogramDataPoint()
			doubleHisto := pdata.NewDoubleHistogramDataPoint()

			for j := 0; j < 5; j++ {
				mb.WithLabels(map[string]string{
					"partition": strconv.Itoa(j),
				}).
					AddGaugeDataPoint("partition.capacity", 40000).
					AddSumDataPoint("disk.reads", 5).
					AddHistogramRawDataPoint("disk.times", intHisto).
					AddDHistogramRawDataPoint("disk.double_times", doubleHisto)
			}
			wg.Done()
		}(strconv.Itoa(i))
	}

	close(ready)
	wg.Wait()

	mCount, dpCount := metrics.MetricAndDataPointCount()
	require.Equal(t, 4004, mCount)
	require.Equal(t, 24000, dpCount)
}

func BenchmarkSimpleMetrics(b *testing.B) {
	for n := 0; n < b.N; n++ {
		mb := Metrics{
			Metrics:                       pdata.NewMetrics(),
			InstrumentationLibraryName:    "example",
			InstrumentationLibraryVersion: "0.1",
			ResourceAttributes: map[string]string{
				"host":    "my-host",
				"service": "app",
			},
			Timestamp: time.Now(),
			Labels: map[string]string{
				"env":     "prod",
				"app":     "myapp",
				"version": "1.0",
			},
		}

		for i := 0; i < 50; i++ {
			name := "gauge" + strconv.Itoa(i)
			mb.AddGaugeDataPoint(name, 5)
			mb.AddGaugeDataPoint(name, 5)
		}
	}
}

func BenchmarkPdataMetrics(b *testing.B) {
	for n := 0; n < b.N; n++ {
		tsNano := pdata.TimestampFromTime(time.Now())

		m := pdata.NewMetrics()

		rms := m.ResourceMetrics()

		rmsLen := rms.Len()
		rms.Resize(rmsLen + 1)
		rm := rms.At(rmsLen)

		res := rm.Resource()
		resAttrs := res.Attributes()
		resAttrs.Insert("host", pdata.NewAttributeValueString("my-host"))
		resAttrs.Insert("serviceName", pdata.NewAttributeValueString("app"))

		ilms := rm.InstrumentationLibraryMetrics()
		ilms.Resize(1)
		ilm := ilms.At(0)
		metrics := ilm.Metrics()
		metrics.Resize(6)

		il := ilm.InstrumentationLibrary()
		il.SetName("example")
		il.SetVersion("0.1")

		for i := 0; i < 50; i++ {
			metric := metrics.At(0)
			metric.SetName("gauge" + strconv.Itoa(i))
			metric.SetDataType(pdata.MetricDataTypeIntGauge)
			mAsType := metric.IntGauge()
			dps := mAsType.DataPoints()
			dps.Resize(2)
			{
				dp := dps.At(0)
				labels := dp.LabelsMap()
				labels.InitEmptyWithCapacity(3)
				labels.Insert("env", "prod")
				labels.Insert("app", "myapp")
				labels.Insert("version", "1.0")
				dp.SetValue(5)
				dp.SetTimestamp(tsNano)
			}
			{
				dp := dps.At(1)
				labels := dp.LabelsMap()
				labels.InitEmptyWithCapacity(3)
				labels.Insert("env", "prod")
				labels.Insert("app", "myapp")
				labels.Insert("version", "1.0")
				dp.SetValue(5)
				dp.SetTimestamp(tsNano)
			}
		}
	}
}
