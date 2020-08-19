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

package regexp

import (
	"regexp"

	"github.com/golang/groupcache/lru"
)

// FilterSet encapsulates a set of filters and caches match results.
// Filters are re2 regex strings.
// FilterSet is exported for convenience, but has unexported fields and should be constructed through NewFilterSet.
//
// FilterSet satisfies the FilterSet interface from
// "go.opentelemetry.io/collector/internal/processor/filterset"
type FilterSet struct {
	regexes      []*regexp.Regexp
	cacheEnabled bool
	cache        *lru.Cache
}

// NewFilterSet constructs a FilterSet of re2 regex strings.
// If any of the given filters fail to compile into re2, an error is returned.
func NewFilterSet(filters []string, cfg *Config) (*FilterSet, error) {
	fs := &FilterSet{
		regexes: make([]*regexp.Regexp, 0, len(filters)),
	}

	if cfg != nil && cfg.CacheEnabled {
		fs.cacheEnabled = true
		fs.cache = lru.New(cfg.CacheMaxNumEntries)
	}

	if err := fs.addFilters(filters); err != nil {
		return nil, err
	}

	return fs, nil
}

// Matches returns true if the given string matches any of the FilterSet's filters.
// The given string must be fully matched by at least one filter's re2 regex.
func (rfs *FilterSet) Matches(toMatch string) bool {
	if rfs.cacheEnabled {
		if v, ok := rfs.cache.Get(toMatch); ok {
			return v.(bool)
		}
	}

	for _, r := range rfs.regexes {
		if r.MatchString(toMatch) {
			if rfs.cacheEnabled {
				rfs.cache.Add(toMatch, true)
			}
			return true
		}
	}

	if rfs.cacheEnabled {
		rfs.cache.Add(toMatch, false)
	}
	return false
}

// addFilters compiles all the given filters and stores them as regexes.
// All regexes are automatically anchored to enforce full string matches.
func (rfs *FilterSet) addFilters(filters []string) error {
	dedup := make(map[string]struct{}, len(filters))
	for _, f := range filters {
		if _, ok := dedup[f]; ok {
			continue
		}

		re, err := regexp.Compile(f)
		if err != nil {
			return err
		}
		rfs.regexes = append(rfs.regexes, re)
		dedup[f] = struct{}{}
	}

	return nil
}
