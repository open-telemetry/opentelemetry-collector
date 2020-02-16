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

package filterset

import (
	"regexp"
	"testing"

	"github.com/golang/groupcache/lru"
	"github.com/stretchr/testify/assert"
)

func TestFactory_CreateFilterSetRegexp(t *testing.T) {
	factory := Factory{}

	tests := []struct {
		name string
		cfg  *FilterConfig
		want *regexpFilterSet
	}{
		{
			name: "emptyConfig",
			cfg: &FilterConfig{
				FilterType: REGEXP,
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: false,
				cache:        nil,
			},
		}, {
			name: "emptyRegexpConfig",
			cfg: &FilterConfig{
				FilterType: REGEXP,
				Regexp:     &RegexpConfig{},
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: false,
				cache:        nil,
			},
		}, {
			name: "ignoreStrictConfig",
			cfg: &FilterConfig{
				FilterType: REGEXP,
				Strict:     &StrictConfig{},
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: false,
				cache:        nil,
			},
		}, {
			name: "cacheDisabledWithSize",
			cfg: &FilterConfig{
				FilterType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled: false,
					CacheSize:    6,
				},
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: false,
				cache:        nil,
			},
		}, {
			name: "cacheEnabledNoSize",
			cfg: &FilterConfig{
				FilterType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled: true,
				},
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: true,
				cache:        lru.New(0), // unlimited cache size
			},
		}, {
			name: "cacheEnabledWithSize",
			cfg: &FilterConfig{
				FilterType: REGEXP,
				Regexp: &RegexpConfig{
					CacheEnabled: true,
					CacheSize:    5,
				},
			},
			want: &regexpFilterSet{
				regexes:      map[string]*regexp.Regexp{},
				cacheEnabled: true,
				cache:        lru.New(5), // unlimited cache size
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := factory.CreateFilterSet(test.cfg)
			assert.NotNil(t, fs)
			assert.Nil(t, err)

			assert.Equal(t, test.want, fs.(*regexpFilterSet))
		})
	}
}

func TestFactory_CreateFilterSetStrict(t *testing.T) {
	factory := Factory{}

	tests := []struct {
		name string
		cfg  *FilterConfig
		want *strictFilterSet
	}{
		{
			name: "strictIsDefault",
			cfg:  &FilterConfig{},
			want: &strictFilterSet{
				filters: map[string]bool{},
			},
		}, {
			name: "emptyStrictConfig",
			cfg: &FilterConfig{
				FilterType: STRICT,
				Strict:     &StrictConfig{},
			},
			want: &strictFilterSet{
				filters: map[string]bool{},
			},
		}, {
			name: "ignoreRegexpConfig",
			cfg: &FilterConfig{
				FilterType: STRICT,
				Regexp:     &RegexpConfig{},
			},
			want: &strictFilterSet{
				filters: map[string]bool{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fs, err := factory.CreateFilterSet(test.cfg)
			assert.NotNil(t, fs)
			assert.Nil(t, err)

			assert.Equal(t, test.want, fs.(*strictFilterSet))
		})
	}
}
