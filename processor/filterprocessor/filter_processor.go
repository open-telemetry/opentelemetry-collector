// Copyright The OpenTelemetry Authors
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

package filterprocessor

import (
	"context"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
)

type filterMetricProcessor struct {
	cfg     *Config
	next    consumer.MetricsConsumer
	include *filtermetric.Matcher
	exclude *filtermetric.Matcher
}

var _ component.MetricsProcessor = (*filterMetricProcessor)(nil)

func newFilterMetricProcessor(next consumer.MetricsConsumer, cfg *Config) (*filterMetricProcessor, error) {
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
		next:    next,
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

// GetCapabilities returns the Capabilities assocciated with the resource processor.
func (fmp *filterMetricProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Start is invoked during service startup.
func (*filterMetricProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*filterMetricProcessor) Shutdown(_ context.Context) error {
	return nil
}

// ConsumeMetricsData implements the MetricsProcessor interface
func (fmp *filterMetricProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return fmp.next.ConsumeMetrics(ctx, fmp.filterMetrics(md))
}

// filterMetrics filters the given spans based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) filterMetrics(md pdata.Metrics) pdata.Metrics {
	mds := pdatautil.MetricsToMetricsData(md)
	for i := range mds {
		if len(mds[i].Metrics) == 0 {
			continue
		}
		keep := make([]*metricspb.Metric, 0, len(mds[i].Metrics))
		for _, m := range mds[i].Metrics {
			if fmp.shouldKeepMetric(m) {
				keep = append(keep, m)
			}
		}
		mds[i].Metrics = keep
	}
	return pdatautil.MetricsFromMetricsData(mds)
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
