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
	"github.com/golang/groupcache/lru"
)

// Option is the type for regexp filtering options that can be passed to NewRegexpFilterSet.
type Option func(*regexpFilterSet)

// WithCache enables an LRU cache that stores the previous results of calls to Matches.
// The cache's max number of entries is set to maxNumEntries. Passing a value of 0 results in an unlimited cache size.
func WithCache(maxNumEntries int) Option {
	return func(rfs *regexpFilterSet) {
		rfs.cacheEnabled = true
		rfs.cache = lru.New(maxNumEntries)
	}
}

// WithFullMatchRequired requires the full string to match one of the regexp filters to be a match for the FilterSet.
// If the regexp pattern matches only a portion of the string, it will be considered a mismatch.
// This is the equivalent of adding the start anchor '^' and end achor '$' to each filter pattern.
//
// Example:
// Filter: "apple" (will be taken as "^apple$")
// Matches: "apple"
// Mismatches: "apples", "sapple"
func WithFullMatchRequired() Option {
	return func(rfs *regexpFilterSet) {
		rfs.fullMatchRequired = true
	}
}
