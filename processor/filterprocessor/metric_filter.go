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

package filterprocessor

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type MetricsFilter struct {
	include filtermetric.Matcher
	exclude filtermetric.Matcher

	logger *zap.Logger
}

// FilterMetrics filters the given metrics based off the MetricsFilter's filters.
func (f *MetricsFilter) FilterMetrics(pdm pdata.Metrics) (pdata.Metrics, error) {
	rms := pdm.ResourceMetrics()
	idx := newMetricIndex()
	for i := 0; i < rms.Len(); i++ {
		ilms := rms.At(i).InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ms := ilms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				keep, err := f.shouldKeepMetric(ms.At(k))
				if err != nil {
					f.logger.Error("shouldKeepMetric failed", zap.Error(err))
					// don't `continue`, keep the metric if there's an error
				}
				if keep {
					idx.add(i, j, k)
				}
			}
		}
	}
	if idx.isEmpty() {
		return pdm, processorhelper.ErrSkipProcessingData
	}
	return idx.extract(pdm), nil
}

func (f *MetricsFilter) shouldKeepMetric(metric pdata.Metric) (bool, error) {
	if f.include != nil {
		matches, err := f.include.MatchMetric(metric)
		if err != nil {
			// default to keep if there's an error
			return true, err
		}
		if !matches {
			return false, nil
		}
	}

	if f.exclude != nil {
		matches, err := f.exclude.MatchMetric(metric)
		if err != nil {
			return true, err
		}
		if matches {
			return false, nil
		}
	}

	return true, nil
}

func NewMetricsFilter(
	include *filtermetric.MatchProperties,
	exclude *filtermetric.MatchProperties,
	logger *zap.Logger) (*MetricsFilter, error) {
	inc, err := createMatcher(include)
	if err != nil {
		return nil, err
	}

	exc, err := createMatcher(exclude)
	if err != nil {
		return nil, err
	}

	return &MetricsFilter{
		include: inc,
		exclude: exc,
		logger:  logger,
	}, nil
}

func createMatcher(mp *filtermetric.MatchProperties) (filtermetric.Matcher, error) {
	// Nothing specified in configuration
	if mp == nil {
		return nil, nil
	}
	return filtermetric.NewMatcher(mp)
}
