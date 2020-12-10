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
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/processor/filterprocessor"
)

type FilterSettings struct {
	filterprocessor.MetricsFilterConfig `mapstructure:"metrics_filter"`
}

// consumerWithFilter wraps a consumer with *filterprocessor.MetricsFilter
type consumerWithFilter struct {
	metricsFilter *filterprocessor.MetricsFilter
	nextConsumer  consumer.MetricsConsumer
}

// ConsumerWithFilter returns the consumer wrapped with a filterCfg. This is an experimental
// feature, expect breaking changes.
func ConsumerWithFilter(
	logger *zap.Logger,
	consumer consumer.MetricsConsumer,
	filterSettings FilterSettings) (consumer.MetricsConsumer, error) {
	metricsFilterer, err := filterprocessor.NewMetricsFilter(filterSettings.Include, filterSettings.Exclude, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics filterCfg: %w", err)
	}

	return &consumerWithFilter{
		metricsFilter: metricsFilterer,
		nextConsumer:  consumer,
	}, nil
}

func (cwf *consumerWithFilter) ConsumeMetrics(ctx context.Context, pdm pdata.Metrics) error {
	filteredPdm, err := cwf.metricsFilter.FilterMetrics(pdm)
	if err != nil {
		return err
	}
	return cwf.nextConsumer.ConsumeMetrics(ctx, filteredPdm)
}
