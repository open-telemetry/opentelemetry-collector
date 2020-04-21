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

package regexp

import (
	"fmt"
	"regexp"

	"github.com/golang/groupcache/lru"

	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
)

// regexpFilterSet encapsulates a set of filters and caches match results.
// Filters are re2 regex strings.
type regexpFilterSet struct {
	regexes           map[string]*regexp.Regexp
	cacheEnabled      bool
	cache             *lru.Cache
	fullMatchRequired bool
}

// NewRegexpFilterSet constructs a FilterSet of re2 regex strings.
// If any of the given filters fail to compile into re2, an error is returned.
func NewRegexpFilterSet(filters []string, opts ...Option) (filterset.FilterSet, error) {
	fs := &regexpFilterSet{
		regexes: map[string]*regexp.Regexp{},
	}

	for _, o := range opts {
		o(fs)
	}

	if err := fs.addFilters(filters); err != nil {
		return nil, err
	}

	return fs, nil
}

// Matches returns true if the given string matches any of the FilterSet's filters.
// The given string must be fully matched by at least one filter's re2 regex.
func (rfs *regexpFilterSet) Matches(toMatch string) bool {
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
func (rfs *regexpFilterSet) addFilters(filters []string) error {
	for _, f := range filters {
		pattern := f
		if rfs.fullMatchRequired {
			// anchor all regexes to enforce full matches
			pattern = fmt.Sprintf("^%s$", f)
		}
		if _, ok := rfs.regexes[pattern]; ok {
			continue
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		rfs.regexes[f] = re
	}

	return nil
}
