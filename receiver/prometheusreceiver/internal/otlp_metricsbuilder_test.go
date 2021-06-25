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

package internal

import (
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestGetBoundaryEquivalence(t *testing.T) {
	cases := []struct {
		name      string
		mtype     metricspb.MetricDescriptor_Type
		pmtype    pdata.MetricDataType
		labels    labels.Labels
		wantValue float64
		wantErr   string
	}{
		{
			name:   "cumulative histogram with bucket label",
			mtype:  metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			pmtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "0.256"},
			},
			wantValue: 0.256,
		},
		{
			name:   "gauge histogram with bucket label",
			mtype:  metricspb.MetricDescriptor_GAUGE_DISTRIBUTION,
			pmtype: pdata.MetricDataTypeIntHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantValue: 11.71,
		},
		{
			name:   "summary with bucket label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:   "summary with quantile label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.QuantileLabel, Value: "92.88"},
			},
			wantValue: 92.88,
		},
		{
			name:   "gauge histogram mismatched with bucket label",
			mtype:  metricspb.MetricDescriptor_SUMMARY,
			pmtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:   "other data types without matches",
			mtype:  metricspb.MetricDescriptor_GAUGE_DOUBLE,
			pmtype: pdata.MetricDataTypeDoubleGauge,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "given metricType has no BucketLabel or QuantileLabel",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			oldBoundary, oerr := getBoundary(tt.mtype, tt.labels)
			pdataBoundary, perr := getBoundaryPdata(tt.pmtype, tt.labels)
			assert.Equal(t, oldBoundary, pdataBoundary, "Both boundary values MUST be equal")
			assert.Equal(t, oldBoundary, tt.wantValue, "Mismatched boundary messages")
			assert.Equal(t, oerr, perr, "The exact same error MUST be returned from both boundary helpers")

			if tt.wantErr != "" {
				require.NotEqual(t, oerr, "expected an error from old style boundary retrieval")
				require.NotEqual(t, perr, "expected an error from new style boundary retrieval")
				require.Contains(t, oerr.Error(), tt.wantErr)
				require.Contains(t, perr.Error(), tt.wantErr)
			}
		})
	}
}

func TestGetBoundaryPdata(t *testing.T) {
	tests := []struct {
		name      string
		mtype     pdata.MetricDataType
		labels    labels.Labels
		wantValue float64
		wantErr   string
	}{
		{
			name:  "cumulative histogram with bucket label",
			mtype: pdata.MetricDataTypeHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "0.256"},
			},
			wantValue: 0.256,
		},
		{
			name:  "gauge histogram with bucket label",
			mtype: pdata.MetricDataTypeIntHistogram,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantValue: 11.71,
		},
		{
			name:  "summary with bucket label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:  "summary with quantile label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.QuantileLabel, Value: "92.88"},
			},
			wantValue: 92.88,
		},
		{
			name:  "gauge histogram mismatched with bucket label",
			mtype: pdata.MetricDataTypeSummary,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "QuantileLabel is empty",
		},
		{
			name:  "other data types without matches",
			mtype: pdata.MetricDataTypeDoubleGauge,
			labels: labels.Labels{
				{Name: model.BucketLabel, Value: "11.71"},
			},
			wantErr: "given metricType has no BucketLabel or QuantileLabel",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			value, err := getBoundaryPdata(tt.mtype, tt.labels)
			if tt.wantErr != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.Nil(t, err)
			require.Equal(t, value, tt.wantValue)
		})
	}
}

func TestConvToPdataMetricType(t *testing.T) {
	tests := []struct {
		name  string
		mtype textparse.MetricType
		want  pdata.MetricDataType
	}{
		{
			name:  "textparse.counter",
			mtype: textparse.MetricTypeCounter,
			want:  pdata.MetricDataTypeDoubleSum,
		},
		{
			name:  "textparse.gauge",
			mtype: textparse.MetricTypeCounter,
			want:  pdata.MetricDataTypeDoubleSum,
		},
		{
			name:  "textparse.unknown",
			mtype: textparse.MetricTypeUnknown,
			want:  pdata.MetricDataTypeDoubleGauge,
		},
		{
			name:  "textparse.histogram",
			mtype: textparse.MetricTypeHistogram,
			want:  pdata.MetricDataTypeHistogram,
		},
		{
			name:  "textparse.summary",
			mtype: textparse.MetricTypeSummary,
			want:  pdata.MetricDataTypeSummary,
		},
		{
			name:  "textparse.metric_type_info",
			mtype: textparse.MetricTypeInfo,
			want:  pdata.MetricDataTypeNone,
		},
		{
			name:  "textparse.metric_state_set",
			mtype: textparse.MetricTypeStateset,
			want:  pdata.MetricDataTypeNone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := convToPdataMetricType(tt.mtype)
			require.Equal(t, got, tt.want)
		})
	}
}

func TestIsusefulLabelPdata(t *testing.T) {
	tests := []struct {
		name      string
		mtypes    []pdata.MetricDataType
		labelKeys []string
		want      bool
	}{
		{
			name: `unuseful "metric","instance","scheme","path","job" with any kind`,
			labelKeys: []string{
				model.MetricNameLabel, model.InstanceLabel, model.SchemeLabel, model.MetricsPathLabel, model.JobLabel,
			},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeDoubleSum,
				pdata.MetricDataTypeDoubleGauge,
				pdata.MetricDataTypeIntHistogram,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeIntSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeIntGauge,
				pdata.MetricDataTypeIntSum,
			},
			want: false,
		},
		{
			name:      `bucket label with "int_histogram", "histogram":: non-useful`,
			mtypes:    []pdata.MetricDataType{pdata.MetricDataTypeIntHistogram, pdata.MetricDataTypeHistogram},
			labelKeys: []string{model.BucketLabel},
			want:      false,
		},
		{
			name: `bucket label with non "int_histogram", "histogram":: useful`,
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeDoubleSum,
				pdata.MetricDataTypeDoubleGauge,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeIntSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeIntGauge,
				pdata.MetricDataTypeIntSum,
			},
			labelKeys: []string{model.BucketLabel},
			want:      true,
		},
		{
			name: `quantile label with "summary": non-useful`,
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeSummary,
			},
			labelKeys: []string{model.QuantileLabel},
			want:      false,
		},
		{
			name:      `quantile label with non-"summary": useful`,
			labelKeys: []string{model.QuantileLabel},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeDoubleSum,
				pdata.MetricDataTypeDoubleGauge,
				pdata.MetricDataTypeIntHistogram,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeIntSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeIntGauge,
				pdata.MetricDataTypeIntSum,
			},
			want: true,
		},
		{
			name:      `any other label with any type:: useful`,
			labelKeys: []string{"any_label", "foo.bar"},
			mtypes: []pdata.MetricDataType{
				pdata.MetricDataTypeDoubleSum,
				pdata.MetricDataTypeDoubleGauge,
				pdata.MetricDataTypeIntHistogram,
				pdata.MetricDataTypeHistogram,
				pdata.MetricDataTypeSummary,
				pdata.MetricDataTypeIntSum,
				pdata.MetricDataTypeNone,
				pdata.MetricDataTypeIntGauge,
				pdata.MetricDataTypeIntSum,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			for _, mtype := range tt.mtypes {
				for _, labelKey := range tt.labelKeys {
					got := isUsefulLabelPdata(mtype, labelKey)
					assert.Equal(t, got, tt.want)
				}
			}
		})
	}
}
