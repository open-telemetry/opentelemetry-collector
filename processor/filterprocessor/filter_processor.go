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

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
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
	mds := pdatautil.MetricsToMetricsData(md)
	foundMetricToKeep := false
	for i := range mds {
		if len(mds[i].Metrics) == 0 {
			continue
		}
		keep := make([]*metricspb.Metric, 0, len(mds[i].Metrics))
		for _, m := range mds[i].Metrics {
			if fmp.shouldKeepMetric(m) {
				foundMetricToKeep = true
				keep = append(keep, m)
			}
		}
		mds[i].Metrics = keep
	}

	if !foundMetricToKeep {
		return md, processorhelper.ErrSkipProcessingData
	}
	return pdatautil.MetricsFromMetricsData(mds), nil
}

// shouldKeepMetric determines whether a metric should be kept based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) shouldKeepMetric(metric *metricspb.Metric) bool {
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
