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

package factory

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/regexp"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset/strict"
)

// Factory can be used to create FilterSets.
type Factory struct{}

// CreateFilterSet creates a FilterSet using cfg.
func (f *Factory) CreateFilterSet(filters []string, cfg *MatchConfig) (filterset.FilterSet, error) {
	switch cfg.MatchType {
	case REGEXP:
		return f.createRegexFilterSet(filters, cfg)
	case STRICT:
		return f.createStrictFilterSet(filters, cfg)
	default:
		return nil, fmt.Errorf("unrecognized filter type: %v", cfg.MatchType)
	}
}

func (f *Factory) createRegexFilterSet(filters []string, cfg *MatchConfig) (filterset.FilterSet, error) {
	if cfg.Regexp != nil && cfg.Regexp.CacheEnabled {
		return regexp.NewRegexpFilterSet(filters, regexp.WithCache(cfg.Regexp.CacheMaxNumEntries))
	}

	return regexp.NewRegexpFilterSet(filters)
}

func (f *Factory) createStrictFilterSet(filters []string, cfg *MatchConfig) (filterset.FilterSet, error) {
	return strict.NewStrictFilterSet(filters)
}
