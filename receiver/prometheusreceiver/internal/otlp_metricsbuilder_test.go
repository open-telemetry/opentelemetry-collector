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

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/pdata"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
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
