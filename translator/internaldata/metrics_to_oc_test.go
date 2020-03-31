// Copyright 2020 OpenTelemetry Authors
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

package internaldata

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestResourceMetricsToMetricsData(t *testing.T) {

	sampleMetricData := testdata.GenerateMetricDataWithCountersHistogramAndSummary()
	attrs := sampleMetricData.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.AttributeHostHostname, "host1"))
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.OCAttributeProcessID, "123"))
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.OCAttributeProcessStartTime, "2020-02-11T20:26:00Z"))
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.AttributeLibraryLanguage, "CPP"))
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.AttributeLibraryVersion, "v2.0.1"))
	attrs.Upsert(data.NewAttributeKeyValueString(conventions.OCAttributeExporterVersion, "v1.2.0"))

	tests := []struct {
		name     string
		internal data.MetricData
		oc       []consumerdata.MetricsData
	}{
		{
			name:     "none",
			internal: data.NewMetricData(),
			oc:       []consumerdata.MetricsData(nil),
		},

		{
			name:     "no-libraries",
			internal: testdata.GenerateMetricDataNoLibraries(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoMetrics(),
			},
		},

		{
			name:     "no-metrics",
			internal: testdata.GenerateMetricDataNoMetrics(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoMetrics(),
			},
		},

		{
			name:     "no-points",
			internal: testdata.GenerateMetricDataAllTypesNoDataPoints(),
			oc: []consumerdata.MetricsData{
				generateOcTestDataNoPoints(),
			},
		},

		{
			name:     "sample-metric",
			internal: sampleMetricData,
			oc: []consumerdata.MetricsData{
				generateOcTestData(t),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := MetricDataToOC(test.internal)
			assert.EqualValues(t, test.oc, got)
		})
	}
}

func generateOcTestDataNoMetrics() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{"resource-attr": "resource-attr-val-1"},
		},
		Metrics: []*ocmetrics.Metric(nil),
	}
}

func generateOcTestDataNoPoints() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{"resource-attr": "resource-attr-val-1"},
		},
		Metrics: []*ocmetrics.Metric{
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "gauge-double",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "gauge-int",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_GAUGE_INT64,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "counter-double",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "counter-int",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "gauge-histogram",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_GAUGE_DISTRIBUTION,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "cumulative-histogram",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
				},
			},
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:        "summary",
					Description: "",
					Unit:        "1",
					Type:        ocmetrics.MetricDescriptor_SUMMARY,
				},
			},
		},
	}
}

func generateOcTestData(t *testing.T) consumerdata.MetricsData {
	ts, err := ptypes.TimestampProto(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))
	assert.NoError(t, err)

	ocMetricInt := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        testdata.TestCounterIntMetricName,
			Description: "",
			Unit:        "1",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "int-label-1"},
				{Key: "int-label-2"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "int-label-value-1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 123,
						},
					},
				},
			},
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						HasValue: false,
					},
					{
						// key2
						Value:    "int-label-value-2",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_Int64Value{
							Int64Value: 456,
						},
					},
				},
			},
		},
	}
	ocMetricDouble := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        testdata.TestCounterDoubleMetricName,
			Description: "",
			Unit:        "1",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE,
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 1.23,
						},
					},
				},
			},
		},
	}
	ocMetricHistogram := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        testdata.TestCumulativeHistogramMetricName,
			Description: "",
			Unit:        "1",
			Type:        ocmetrics.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "histogram-label-1"},
				{Key: "histogram-label-2"},
				{Key: "histogram-label-3"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "histogram-label-value-1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						Value:    "histogram-label-value-3",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_DistributionValue{
							DistributionValue: &ocmetrics.DistributionValue{
								Count: 1,
								Sum:   15,
							},
						},
					},
				},
			},
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						HasValue: false,
					},
					{
						// key2
						Value:    "histogram-label-value-2",
						HasValue: true,
					},
					{
						// key3
						HasValue: false,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_DistributionValue{
							DistributionValue: &ocmetrics.DistributionValue{
								Count: 1,
								Sum:   15,
								BucketOptions: &ocmetrics.DistributionValue_BucketOptions{
									Type: &ocmetrics.DistributionValue_BucketOptions_Explicit_{
										Explicit: &ocmetrics.DistributionValue_BucketOptions_Explicit{
											Bounds: []float64{1},
										},
									},
								},
								Buckets: []*ocmetrics.DistributionValue_Bucket{
									{
										Count: 0,
									},
									{
										Count: 1,
										Exemplar: &ocmetrics.DistributionValue_Exemplar{
											Timestamp:   internal.TimeToTimestamp(testdata.TestMetricExemplarTime),
											Value:       15,
											Attachments: map[string]string{"exemplar-attachment": "exemplar-attachment-value"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	ocMetricSummary := &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name:        testdata.TestSummaryMetricName,
			Description: "",
			Unit:        "1",
			Type:        ocmetrics.MetricDescriptor_SUMMARY,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "summary-label"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "summary-label-value-1",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_SummaryValue{
							SummaryValue: &ocmetrics.SummaryValue{
								Count: &wrappers.Int64Value{
									Value: 1,
								},
								Sum: &wrappers.DoubleValue{
									Value: 15,
								},
							},
						},
					},
				},
			},
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "summary-label-value-2",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_SummaryValue{
							SummaryValue: &ocmetrics.SummaryValue{
								Count: &wrappers.Int64Value{
									Value: 1,
								},
								Sum: &wrappers.DoubleValue{
									Value: 15,
								},
								Snapshot: &ocmetrics.SummaryValue_Snapshot{
									PercentileValues: []*ocmetrics.SummaryValue_Snapshot_ValueAtPercentile{
										{
											Percentile: 1,
											Value:      15,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return consumerdata.MetricsData{
		Node: &occommon.Node{
			Identifier: &occommon.ProcessIdentifier{
				HostName:       "host1",
				Pid:            123,
				StartTimestamp: ts,
			},
			LibraryInfo: &occommon.LibraryInfo{
				Language:           occommon.LibraryInfo_CPP,
				ExporterVersion:    "v1.2.0",
				CoreLibraryVersion: "v2.0.1",
			},
		},
		Resource: &ocresource.Resource{
			Labels: map[string]string{
				"resource-attr": "resource-attr-val-1",
			},
		},
		Metrics: []*ocmetrics.Metric{ocMetricInt, ocMetricDouble, ocMetricHistogram, ocMetricSummary},
	}
}
