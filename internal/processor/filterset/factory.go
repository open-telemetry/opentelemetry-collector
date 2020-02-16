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

package filterset

// FilterConfig can be used to configure a filter.
type FilterConfig struct {
	FilterType FilterType    `mapstructure:"match_type"`
	Filters    []string      `mapstructure:"matches"`
	Regexp     *RegexpConfig `mapstructure:"regexp"`
	Strict     *StrictConfig `mapstructure:"strict"`
}

// StrictConfig is used to configure options for NewStrictFilterSet.
type StrictConfig struct {
}

// RegexpConfig represents the options for a NewRegexpFilterSet.
type RegexpConfig struct {
	// CacheEnabled determines whether match results are LRU cached to
	// make subsequent matches faster. Cache size is unlimited unless
	// CacheSize is also specified.
	CacheEnabled bool `mapstructure:"cacheenabled"`
	// CacheSize is the size of the LRU cache that stores match results.
	// CacheSize is ignored if CacheEnabled is false.
	CacheSize int `mapstructure:"cachesize"`
}

// Factory can be used to create FilterSets.
type Factory struct{}

// CreateFilterSet creates a FilterSet using cfg.
func (f *Factory) CreateFilterSet(cfg *FilterConfig) (FilterSet, error) {
	switch cfg.FilterType {
	case REGEXP:
		return f.createRegexFilterSet(cfg)
	case STRICT:
		return f.createStrictFilterSet(cfg)
	default:
		return f.createStrictFilterSet(cfg)
	}
}

func (f *Factory) createRegexFilterSet(cfg *FilterConfig) (FilterSet, error) {
	if cfg.Regexp != nil {
		if cfg.Regexp.CacheEnabled {
			return NewRegexpFilterSet(cfg.Filters, WithCacheSize(cfg.Regexp.CacheSize))
		}
	}

	return NewRegexpFilterSet(cfg.Filters)
}

func (f *Factory) createStrictFilterSet(cfg *FilterConfig) (FilterSet, error) {
	return NewStrictFilterSet(cfg.Filters)
}
