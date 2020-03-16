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

package metric

import (
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/factory"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
)

func createMetric(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
		},
	}
}

func createConfig(filters []string, matchType factory.MatchType) *MatchProperties {
	return &MatchProperties{
		MatchConfig: factory.MatchConfig{
			MatchType: matchType,
		},
		MetricNames: filters,
	}
}

func createConfigWithRegexpOptions(filters []string, rCfg *factory.RegexpConfig) *MatchProperties {
	cfg := createConfig(filters, factory.REGEXP)
	cfg.MatchConfig.Regexp = rCfg
	return cfg
}

func createConfigWithStrictOptions(filters []string, sCfg *factory.StrictConfig) *MatchProperties {
	cfg := createConfig(filters, factory.STRICT)
	cfg.MatchConfig.Strict = sCfg
	return cfg
}
