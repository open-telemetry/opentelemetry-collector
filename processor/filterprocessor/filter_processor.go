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
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type filterMetricProcessor struct {
	cfg             *Config
	metricsFilterer *MetricsFilterer
}

func newFilterMetricProcessor(logger *zap.Logger, cfg *Config) (*filterMetricProcessor, error) {
	includeMatchType := ""
	var includeExpressions []string
	var includeMetricNames []string
	if cfg.Metrics.Include != nil {
		includeMatchType = string(cfg.Metrics.Include.MatchType)
		includeExpressions = cfg.Metrics.Include.Expressions
		includeMetricNames = cfg.Metrics.Include.MetricNames
	}

	excludeMatchType := ""
	var excludeExpressions []string
	var excludeMetricNames []string
	if cfg.Metrics.Exclude != nil {
		excludeMatchType = string(cfg.Metrics.Exclude.MatchType)
		excludeExpressions = cfg.Metrics.Exclude.Expressions
		excludeMetricNames = cfg.Metrics.Exclude.MetricNames
	}

	logger.Info(
		"Metric filter configured",
		zap.String("include match_type", includeMatchType),
		zap.Strings("include expressions", includeExpressions),
		zap.Strings("include metric names", includeMetricNames),
		zap.String("exclude match_type", excludeMatchType),
		zap.Strings("exclude expressions", excludeExpressions),
		zap.Strings("exclude metric names", excludeMetricNames),
	)

	metricsFilterer, err := NewMetricsFilterer(cfg.Metrics.Include, cfg.Metrics.Exclude, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics filter: %w", err)
	}

	return &filterMetricProcessor{
		cfg:             cfg,
		metricsFilterer: metricsFilterer,
	}, nil
}

// ProcessMetrics filters the given metrics based off the filterMetricProcessor's filters.
func (fmp *filterMetricProcessor) ProcessMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	return fmp.metricsFilterer.FilterMetrics(pdm)
}
