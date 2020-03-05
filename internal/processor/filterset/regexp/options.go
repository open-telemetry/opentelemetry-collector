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

// RFSOption is the type for regexp filtering options that can be passed to NewRegexpFilterSet.
type RFSOption func(*regexpFilterSet)

// WithCacheSize sets the regexp filterset's internal cache size to the given size.
// The cache stores the results of previous calls to Matches.
func WithCacheSize(size int) RFSOption {
	return func(rfs *regexpFilterSet) {
		rfs.cacheEnabled = true
		rfs.cache = lru.New(size)
	}
}
