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

const (
	// REGEXP is the FilterType for filtering by regexp string matches.
	REGEXP MatchType = "regexp"
	// STRICT is the FilterType for filtering by exact string matches.
	STRICT MatchType = "strict"
)

// MatchType describes the type of pattern matching a FilterSet uses to filter strings.
type MatchType string

// MatchConfig configures the matching behavior of a FilterSet.
type MatchConfig struct {
	MatchType MatchType     `mapstructure:"match_type"`
	Regexp    *RegexpConfig `mapstructure:"regexp"`
	Strict    *StrictConfig `mapstructure:"strict"`
}

// StrictConfig is used to configure options for NewStrictFilterSet.
type StrictConfig struct{}

// RegexpConfig represents the options for a NewRegexpFilterSet.
type RegexpConfig struct {
	// CacheEnabled determines whether match results are LRU cached to make subsequent matches faster.
	// Cache size is unlimited unless CacheMaxNumEntries is also specified.
	CacheEnabled bool `mapstructure:"cacheenabled"`
	// CacheMaxNumEntries is the max number of entries of the LRU cache that stores match results.
	// CacheMaxNumEntries is ignored if CacheEnabled is false.
	CacheMaxNumEntries int `mapstructure:"cachemaxnumentries"`

	// FullMatchRequired requires the full string to match one of the regexp filters to be a match for the FilterSet.
	// If the regexp pattern matches only a portion of the string, it will be considered a mismatch.
	// This is the equivalent of adding the start anchor '^' and end achor '$' to each filter pattern.
	//
	// Example:
	// Filter: "apple" (will be taken as "^apple$")
	// Matches: "apple"
	// Mismatches: "apples", "sapple"
	FullMatchRequired bool `mapstructure:"fullmatchrequired"`
}
