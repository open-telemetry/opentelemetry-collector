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

package filterprocessor

import (
	"context"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type filterMetricProcessor struct {
	nameFilters  filterset.FilterSet
	cfg          *Config
	capabilities processor.Capabilities
	next         consumer.MetricsConsumer
}

func newFilterMetricProcessor(next consumer.MetricsConsumer, cfg *Config) (*filterMetricProcessor, error) {
	factory := filterset.Factory{}
	nf, err := factory.CreateFilterSet(&cfg.Metrics.NameFilter)
	if err != nil {
		return nil, err
	}

	return &filterMetricProcessor{
		nameFilters:  nf,
		cfg:          cfg,
		capabilities: processor.Capabilities{MutatesConsumedData: true},
		next:         next,
	}, nil
}

// GetCapabilities returns the Capabilities assocciated with the resource processor.
func (fmp *filterMetricProcessor) GetCapabilities() processor.Capabilities {
	return fmp.capabilities
}

// Start is invoked during service startup.
func (*filterMetricProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (*filterMetricProcessor) Shutdown() error {
	return nil
}

// ConsumeMetricsData implements the MetricsProcessor interface
func (fmp *filterMetricProcessor) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return fmp.next.ConsumeMetricsData(ctx, consumerdata.MetricsData{
		Node:     md.Node,
		Resource: md.Resource,
		Metrics:  fmp.filterMetrics(md.Metrics),
	})
}

// filterMetrics filters the given spans based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) filterMetrics(metrics []*metricspb.Metric) []*metricspb.Metric {
	keep := []*metricspb.Metric{}
	for _, m := range metrics {
		if fmp.shouldKeepMetric(m) {
			keep = append(keep, m)
		}
	}

	return keep
}

// shouldKeepMetric determines whether or not a span should be kept based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) shouldKeepMetric(metric *metricspb.Metric) bool {
	name := metric.GetMetricDescriptor().GetName()
	nameMatch := fmp.nameFilters.Matches(name)
	return nameMatch == (fmp.cfg.Action == INCLUDE)
}
