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
			errorString: "at least one of \"log_names\" or \"attributes\" field must be specified",
		},
		{
			name: "empty_log_names_and_attributes",
			property: filterconfig.MatchProperties{
				LogNames: []string{},
			},
			errorString: "at least one of \"log_names\" or \"attributes\" field must be specified",
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
			name: "regexp_match_type_for_attributes",
			property: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				Attributes: []filterconfig.Attribute{
					{Key: "key", Value: "value"},
				},
			},
			errorString: `match_type=regexp is not supported for "attributes"`,
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
		{
			name: "empty_key_name_in_attributes_list",
			property: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key: "",
					},
				},
			},
			errorString: "error creating processor. Can't have empty key in the list of attributes",
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

		{
			name: "wrong_property_value",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: 1234,
					},
				},
			},
		},
		{
			name: "incompatible_property_value",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: "123",
					},
				},
			},
		},
		{
			name: "property_key_does_not_exist",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "doesnotexist",
						Value: nil,
					},
				},
			},
		},
	}

	lr := pdata.NewLogRecord()
	lr.InitEmpty()
	lr.SetName("logName")
	lr.Attributes().InitFromMap(map[string]pdata.AttributeValue{"keyInt": pdata.NewAttributeValueInt(123)})
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, matcher)

			assert.False(t, matcher.MatchLogRecord(lr))
		})
	}
}

func TestLogRecord_MatchingCornerCases(t *testing.T) {
	cfg := &filterconfig.MatchProperties{
		Config:   *createConfig(filterset.Strict),
		LogNames: []string{"svcA"},
		Attributes: []filterconfig.Attribute{
			{
				Key:   "keyOne",
				Value: nil,
			},
		},
	}

	mp, err := NewMatcher(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, mp)

	emptyLogRecord := pdata.NewLogRecord()
	emptyLogRecord.InitEmpty()
	assert.False(t, mp.MatchLogRecord(emptyLogRecord))

	emptyLogRecord.SetName("svcA")
	assert.False(t, mp.MatchLogRecord(emptyLogRecord))
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
		{
			name: "property_exact_value_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyString",
						Value: "arithmetic",
					},
					{
						Key:   "keyInt",
						Value: 123,
					},
					{
						Key:   "keyDouble",
						Value: 3245.6,
					},
					{
						Key:   "keyBool",
						Value: true,
					},
				},
			},
		},
		{
			name: "property_exists",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyExists",
						Value: nil,
					},
				},
			},
		},
		{
			name: "match_all_settings_exists",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				LogNames: []string{"logName"},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyExists",
						Value: nil,
					},
					{
						Key:   "keyString",
						Value: "arithmetic",
					},
				},
			},
		},
	}

	lr := pdata.NewLogRecord()
	lr.InitEmpty()
	lr.SetName("logName")
	lr.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		"keyString": pdata.NewAttributeValueString("arithmetic"),
		"keyInt":    pdata.NewAttributeValueInt(123),
		"keyDouble": pdata.NewAttributeValueDouble(3245.6),
		"keyBool":   pdata.NewAttributeValueBool(true),
		"keyExists": pdata.NewAttributeValueString("present"),
	})

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mp, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, mp)

			assert.NotNil(t, lr)
			assert.True(t, mp.MatchLogRecord(lr))
		})
	}
}

func TestLogRecord_validateMatchesConfigurationForAttributes(t *testing.T) {
	testcase := []struct {
		name   string
		input  filterconfig.MatchProperties
		output Matcher
	}{
		{
			name: "attributes_build",
			input: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &propertiesMatcher{
				Attributes: []attributeMatcher{
					{
						Key: "key1",
					},
					{
						Key:            "key2",
						AttributeValue: newAttributeValueInt(1234),
					},
				},
			},
		},

		{
			name: "both_set_of_attributes",
			input: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &propertiesMatcher{
				Attributes: []attributeMatcher{
					{
						Key: "key1",
					},
					{
						Key:            "key2",
						AttributeValue: newAttributeValueInt(1234),
					},
				},
			},
		},
	}
	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.output, output)
		})
	}
}

func newAttributeValueInt(v int64) *pdata.AttributeValue {
	attr := pdata.NewAttributeValueInt(v)
	return &attr
}
