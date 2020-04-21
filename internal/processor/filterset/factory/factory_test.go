// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactory_CreateFilterSetInvalid(t *testing.T) {
	factory := Factory{}
	tests := []struct {
		name string
		cfg  *MatchConfig
	}{
		{
			name: "invalidFilterType",
			cfg: &MatchConfig{
				MatchType: "INVALID",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := factory.CreateFilterSet([]string{}, test.cfg)
			assert.Nil(t, fs)
			assert.NotNil(t, err)
		})
	}
}

func TestFactory_CreateFilterSetRegexp(t *testing.T) {
	factory := Factory{}
	tests := []struct {
		name string
		cfg  *MatchConfig
	}{
		{
			name: "emptyConfig",
			cfg: &MatchConfig{
				MatchType: REGEXP,
			},
		}, {
			name: "nilRegexpConfig",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp:    nil,
			},
		}, {
			name: "emptyRegexpConfig",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp:    &RegexpConfig{},
			},
		}, {
			name: "ignoreStrictConfig",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Strict:    &StrictConfig{},
			},
		}, {
			name: "cacheDisabledWithSize",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled:       false,
					CacheMaxNumEntries: 6,
				},
			},
		}, {
			name: "cacheEnabledNoSize",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled: true,
				},
			},
		}, {
			name: "cacheEnabledWithSize",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled:       true,
					CacheMaxNumEntries: 5,
				},
			},
		}, {
			name: "withFulLMatchRequired",
			cfg: &MatchConfig{
				MatchType: REGEXP,
				Regexp: &RegexpConfig{
					FullMatchRequired: true,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters := []string{
				"match.*",
			}

			fs, err := factory.CreateFilterSet(filters, test.cfg)
			assert.NotNil(t, fs)
			assert.Nil(t, err)

			assert.True(t, fs.Matches("match_string"))
			assert.False(t, fs.Matches("mis_string"))
		})
	}
}

func TestFactory_CreateFilterSetStrict(t *testing.T) {
	factory := Factory{}

	tests := []struct {
		name string
		cfg  *MatchConfig
	}{
		{
			name: "emptyStrictConfig",
			cfg: &MatchConfig{
				MatchType: STRICT,
				Strict:    &StrictConfig{},
			},
		}, {
			name: "nilStrictConfig",
			cfg: &MatchConfig{
				MatchType: STRICT,
				Strict:    nil,
			},
		}, {
			name: "ignoreRegexpConfig",
			cfg: &MatchConfig{
				MatchType: STRICT,
				Regexp:    &RegexpConfig{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters := []string{
				"match",
			}

			fs, err := factory.CreateFilterSet(filters, test.cfg)
			assert.NotNil(t, fs)
			assert.Nil(t, err)

			assert.True(t, fs.Matches("match"))
			assert.False(t, fs.Matches("mismatch"))
		})
	}
}
