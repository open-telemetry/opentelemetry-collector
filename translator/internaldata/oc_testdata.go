// Copyright 2019 OpenTelemetry Authors
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
	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
)

func generateOCTestDataNoMetrics() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{"resource-attr": "resource-attr-val-1"},
		},
		Metrics: []*ocmetrics.Metric(nil),
	}
}

func generateOCTestDataNoPoints() consumerdata.MetricsData {
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

func generateOCTestDataNoLabels() consumerdata.MetricsData {
	return consumerdata.MetricsData{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{"resource-attr": "resource-attr-val-1"},
		},
		Metrics: []*ocmetrics.Metric{
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name: "counter-int",
					Unit: "1",
					Type: ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
				},
				Timeseries: []*ocmetrics.TimeSeries{
					{
						StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
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
			},
		},
	}
}

func generateOCTestMetricInt() *ocmetrics.Metric {
	return &ocmetrics.Metric{
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
}

func generateOCTestMetricDouble() *ocmetrics.Metric {
	return &ocmetrics.Metric{
		MetricDescriptor: &ocmetrics.MetricDescriptor{
			Name: "counter-double",
			Unit: "1",
			Type: ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE,
			LabelKeys: []*ocmetrics.LabelKey{
				{Key: "double-label-1"},
				{Key: "double-label-2"},
				{Key: "double-label-3"},
			},
		},
		Timeseries: []*ocmetrics.TimeSeries{
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "double-label-value-1",
						HasValue: true,
					},
					{
						// key2
						Value:    "double-label-value-2",
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
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 1.23,
						},
					},
				},
			},
			{
				StartTimestamp: internal.TimeToTimestamp(testdata.TestMetricStartTime),
				LabelValues: []*ocmetrics.LabelValue{
					{
						// key1
						Value:    "double-label-different-value-1",
						HasValue: true,
					},
					{
						// key2
						HasValue: false,
					},
					{
						// key3
						Value:    "double-label-value-3",
						HasValue: true,
					},
				},
				Points: []*ocmetrics.Point{
					{
						Timestamp: internal.TimeToTimestamp(testdata.TestMetricTime),
						Value: &ocmetrics.Point_DoubleValue{
							DoubleValue: 4.56,
						},
					},
				},
			},
		},
	}
}

func generateOCTestMetricHistogram() *ocmetrics.Metric {
	return &ocmetrics.Metric{
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
}

func generateOCTestMetricSummary() *ocmetrics.Metric {
	return &ocmetrics.Metric{
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
}
