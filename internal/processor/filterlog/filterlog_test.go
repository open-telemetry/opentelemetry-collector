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

package filterlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

func createConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}

func TestLogRecord_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
	testcases := []struct {
		name        string
		property    filterconfig.MatchProperties
		errorString string
	}{
		{
			name:        "empty_property",
			property:    filterconfig.MatchProperties{},
			errorString: "at least one of \"log_names\", \"attributes\", \"libraries\" or \"resources\" field must be specified",
		},
		{
			name: "empty_log_names_and_attributes",
			property: filterconfig.MatchProperties{
				LogNames: []string{},
			},
			errorString: "at least one of \"log_names\", \"attributes\", \"libraries\" or \"resources\" field must be specified",
		},
		{
			name: "span_properties",
			property: filterconfig.MatchProperties{
				SpanNames: []string{"span"},
			},
			errorString: "neither services nor span_names should be specified for log records",
		},
		{
			name: "invalid_match_type",
			property: filterconfig.MatchProperties{
				Config:   *createConfig("wrong_match_type"),
				LogNames: []string{"abc"},
			},
			errorString: "error creating log record name filters: unrecognized match_type: 'wrong_match_type', valid types are: [regexp strict]",
		},
		{
			name: "missing_match_type",
			property: filterconfig.MatchProperties{
				LogNames: []string{"abc"},
			},
			errorString: "error creating log record name filters: unrecognized match_type: '', valid types are: [regexp strict]",
		},
		{
			name: "invalid_regexp_pattern",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				LogNames: []string{"["},
			},
			errorString: "error creating log record name filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern2",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				LogNames: []string{"["},
			},
			errorString: "error creating log record name filters: error parsing regexp: missing closing ]: `[`",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.property)
			assert.Nil(t, output)
			require.NotNil(t, err)
			assert.Equal(t, tc.errorString, err.Error())
		})
	}
}

func TestLogRecord_Matching_False(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "log_name_doesnt_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				LogNames:   []string{"logNo.*Name"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "log_name_doesnt_match_any",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				LogNames: []string{
					"logNo.*Name",
					"non-matching?pattern",
					"regular string",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},
	}

	lr := pdata.NewLogRecord()
	lr.SetName("logName")
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, matcher)

			assert.False(t, matcher.MatchLogRecord(lr, pdata.Resource{}, pdata.InstrumentationLibrary{}))
		})
	}
}

func TestLogRecord_Matching_True(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "log_name_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				LogNames:   []string{"log.*"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "log_name_second_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				LogNames: []string{
					"wrong.*pattern",
					"log.*",
					"yet another?pattern",
					"regularstring",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},
	}

	lr := pdata.NewLogRecord()
	lr.SetName("logName")

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mp, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, mp)

			assert.NotNil(t, lr)
			assert.True(t, mp.MatchLogRecord(lr, pdata.Resource{}, pdata.InstrumentationLibrary{}))
		})
	}
}
