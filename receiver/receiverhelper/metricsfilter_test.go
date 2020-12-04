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

package receiverhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/processor/filterprocessor"
)

func TestConsumerWithFilter(t *testing.T) {
	inputMetrics := func() pdata.Metrics {
		out := pdata.NewMetrics()
		rm := out.ResourceMetrics()
		rm.Resize(1)
		ilm := rm.At(0).InstrumentationLibraryMetrics()
		ilm.Resize(1)
		metrics := ilm.At(0).Metrics()
		metrics.Resize(3)
		metrics.At(0).SetName("metric_1")
		metrics.At(1).SetName("metric_2")
		metrics.At(2).SetName("metric_3")
		return out
	}()

	tests := []struct {
		name                    string
		consumer                *consumertest.MetricsSink
		filterSettings          FilterSettings
		inputMetrics            pdata.Metrics
		expectedMetrics         []string
		wantErr                 bool
		wantErrOnConsumeMetrics bool
	}{
		{
			name:           "Empty filter",
			consumer:       &consumertest.MetricsSink{},
			filterSettings: FilterSettings{},
			expectedMetrics: []string{
				"metric_1",
				"metric_2",
				"metric_3",
			},
			inputMetrics: inputMetrics,
		},
		{
			name:     "Strict include filter",
			consumer: &consumertest.MetricsSink{},
			filterSettings: FilterSettings{
				MetricFilters: filterprocessor.MetricFilters{
					Include: &filtermetric.MatchProperties{
						MatchType: "strict",
						MetricNames: []string{
							"metric_1",
						},
					},
				},
			},
			expectedMetrics: []string{
				"metric_1",
			},
			inputMetrics: inputMetrics,
		},
		{
			name:     "Strict exclude filter",
			consumer: &consumertest.MetricsSink{},
			filterSettings: FilterSettings{
				MetricFilters: filterprocessor.MetricFilters{
					Exclude: &filtermetric.MatchProperties{
						MatchType: "strict",
						MetricNames: []string{
							"metric_2",
							"metric_3",
						},
					},
				},
			},
			expectedMetrics: []string{
				"metric_1",
			},
			inputMetrics: inputMetrics,
		},
		{
			name:     "Regex include filter",
			consumer: &consumertest.MetricsSink{},
			filterSettings: FilterSettings{
				MetricFilters: filterprocessor.MetricFilters{
					Include: &filtermetric.MatchProperties{
						MatchType: "regexp",
						MetricNames: []string{
							".*1",
						},
					},
				},
			},
			expectedMetrics: []string{
				"metric_1",
			},
			inputMetrics: inputMetrics,
		},
		{
			name:     "Regex exclude filter",
			consumer: &consumertest.MetricsSink{},
			filterSettings: FilterSettings{
				MetricFilters: filterprocessor.MetricFilters{
					Exclude: &filtermetric.MatchProperties{
						MatchType: "regexp",
						MetricNames: []string{
							".*2",
						},
					},
				},
			},
			expectedMetrics: []string{
				"metric_1",
				"metric_3",
			},
			inputMetrics: inputMetrics,
		},
		{
			name:     "Filter error",
			consumer: &consumertest.MetricsSink{},
			filterSettings: FilterSettings{
				MetricFilters: filterprocessor.MetricFilters{
					Exclude: &filtermetric.MatchProperties{
						MatchType: "regexp",
						MetricNames: []string{
							"*2",
						},
					},
				},
			},
			inputMetrics: inputMetrics,
			wantErr:      true,
		},
		{
			name:                    "ConsumeMetrics error",
			consumer:                &consumertest.MetricsSink{},
			filterSettings:          FilterSettings{},
			wantErrOnConsumeMetrics: true,
			inputMetrics:            pdata.NewMetrics(),
		},
	}

	logger := zap.NewNop()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConsumerWithFilter(logger, tt.consumer, tt.filterSettings)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			err = got.ConsumeMetrics(context.Background(), tt.inputMetrics)
			if tt.wantErrOnConsumeMetrics {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tt.expectedMetrics), tt.consumer.MetricsCount())
			rms := tt.consumer.AllMetrics()[0]
			ilms := rms.ResourceMetrics().At(0).InstrumentationLibraryMetrics()
			for i := 0; i < len(tt.expectedMetrics); i++ {
				metric := ilms.At(0).Metrics().At(i)
				require.Equal(t, tt.expectedMetrics[i], metric.Name())
			}
		})
	}
}
