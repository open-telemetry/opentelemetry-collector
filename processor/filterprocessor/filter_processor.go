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
	"context"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type filterMetricProcessor struct {
	cfg     *Config
	include *filtermetric.Matcher
	exclude *filtermetric.Matcher
}

func newFilterMetricProcessor(cfg *Config) (*filterMetricProcessor, error) {
	inc, err := createMatcher(cfg.Metrics.Include)
	if err != nil {
		return nil, err
	}

	exc, err := createMatcher(cfg.Metrics.Exclude)
	if err != nil {
		return nil, err
	}

	return &filterMetricProcessor{
		cfg:     cfg,
		include: inc,
		exclude: exc,
	}, nil
}

func createMatcher(mp *filtermetric.MatchProperties) (*filtermetric.Matcher, error) {
	// Nothing specified in configuration
	if mp == nil {
		return nil, nil
	}

	matcher, err := filtermetric.NewMatcher(mp)
	if err != nil {
		return nil, err
	}

	return &matcher, nil
}

// ProcessMetrics filters the given metrics based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) ProcessMetrics(_ context.Context, md pdata.Metrics) (pdata.Metrics, error) {
	rms := md.ResourceMetrics()
	foundMetricToKeep := false
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		ilms := rm.InstrumentationLibraryMetrics()
		for j := 0; j < ilms.Len(); j++ {
			ilm := ilms.At(j)
			ms := ilm.Metrics()
			filteredMetrics := pdata.NewMetricSlice()
			for k := 0; k < ms.Len(); k++ {
				metric := ms.At(k)
				if fmp.shouldKeepMetric(metric) {
					foundMetricToKeep = true
					filteredMetrics.Append(metric)
				}
			}
			filteredMetrics.CopyTo(ilm.Metrics())
		}
	}
	if !foundMetricToKeep {
		return md, processorhelper.ErrSkipProcessingData
	}
	return md, nil
}

func (fmp *filterMetricProcessor) shouldKeepMetric(metric pdata.Metric) bool {
	if fmp.include != nil {
		if !fmp.include.MatchMetric(metric) {
			return false
		}
	}

	if fmp.exclude != nil {
		if fmp.exclude.MatchMetric(metric) {
			return false
		}
	}

	return true
}
