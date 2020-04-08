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

package filterspan

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

func TestSpan_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
	testcases := []struct {
		name        string
		property    MatchProperties
		errorString string
	}{
		{
			name:        "empty_property",
			property:    MatchProperties{},
			errorString: errAtLeastOneMatchFieldNeeded.Error(),
		},
		{
			name: "empty_service_span_names_and_attributes",
			property: MatchProperties{
				Services:   []string{},
				Attributes: []Attribute{},
			},
			errorString: errAtLeastOneMatchFieldNeeded.Error(),
		},
		{
			name: "invalid_match_type",
			property: MatchProperties{
				MatchType: MatchType("wrong_match_type"),
				Services:  []string{"abc"},
			},
			errorString: errInvalidMatchType.Error(),
		},
		{
			name: "missing_match_type",
			property: MatchProperties{
				Services: []string{"abc"},
			},
			errorString: errInvalidMatchType.Error(),
		},
		{
			name: "regexp_match_type_for_attributes",
			property: MatchProperties{
				MatchType: MatchTypeRegexp,
				Attributes: []Attribute{
					{Key: "key", Value: "value"},
				},
			},
			errorString: `match_type=regexp is not supported for "attributes"`,
		},
		{
			name: "invalid_regexp_pattern",
			property: MatchProperties{
				MatchType: MatchTypeRegexp,
				Services:  []string{"["},
			},
			errorString: "error creating processor. [ is not a valid service name regexp pattern",
		},
		{
			name: "invalid_regexp_pattern2",
			property: MatchProperties{
				MatchType: MatchTypeRegexp,
				SpanNames: []string{"["},
			},
			errorString: "error creating processor. [ is not a valid span name regexp pattern",
		},
		{
			name: "empty_key_name_in_attributes_list",
			property: MatchProperties{
				MatchType: MatchTypeStrict,
				Services:  []string{"a"},
				Attributes: []Attribute{
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

func TestSpan_Matching_False(t *testing.T) {
	testcases := []struct {
		name       string
		properties Matcher
	}{
		{
			name: "service_name_doesnt_match_regexp",
			properties: &regexpPropertiesMatcher{
				Services:   []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []attributeMatcher{},
			},
		},

		{
			name: "service_name_doesnt_match_strict",
			properties: &strictPropertiesMatcher{
				Services:   []string{"svcA"},
				Attributes: []attributeMatcher{},
			},
		},

		{
			name: "span_name_doesnt_match",
			properties: &regexpPropertiesMatcher{
				SpanNames:  []*regexp.Regexp{regexp.MustCompile("spanNo.*Name")},
				Attributes: []attributeMatcher{},
			},
		},

		{
			name: "span_name_doesnt_match_any",
			properties: &regexpPropertiesMatcher{
				SpanNames: []*regexp.Regexp{
					regexp.MustCompile("spanNo.*Name"),
					regexp.MustCompile("non-matching?pattern"),
					regexp.MustCompile("regular string"),
				},
				Attributes: []attributeMatcher{},
			},
		},

		{
			name: "wrong_property_value",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{},
				Attributes: []attributeMatcher{
					{
						Key:            "keyInt",
						AttributeValue: newAttributeValueInt(1234),
					},
				},
			},
		},
		{
			name: "incompatible_property_value",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{},
				Attributes: []attributeMatcher{
					{
						Key:            "keyInt",
						AttributeValue: newAttributeValueString("123"),
					},
				},
			},
		},
		{
			name: "property_key_does_not_exist",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{},
				Attributes: []attributeMatcher{
					{
						Key:            "doesnotexist",
						AttributeValue: nil,
					},
				},
			},
		},
	}

	span := data.NewSpan()
	span.InitEmpty()
	span.SetName("spanName")
	span.Attributes().InitFromMap(map[string]data.AttributeValue{"keyInt": data.NewAttributeValueInt(123)})
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, tc.properties.MatchSpan(span, "wrongSvc"))
		})
	}
}

func TestSpan_MatchingCornerCases(t *testing.T) {
	mp := &regexpPropertiesMatcher{
		Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
		Attributes: []attributeMatcher{
			{
				Key:            "keyOne",
				AttributeValue: nil,
			},
		},
	}
	emptySpan := data.NewSpan()
	emptySpan.InitEmpty()
	assert.False(t, mp.MatchSpan(emptySpan, "svcA"))
}

func TestSpan_MissingServiceName(t *testing.T) {
	mp := &regexpPropertiesMatcher{
		Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
	}

	emptySpan := data.NewSpan()
	emptySpan.InitEmpty()
	assert.False(t, mp.MatchSpan(emptySpan, ""))
}

func TestSpan_Matching_True(t *testing.T) {
	testcases := []struct {
		name       string
		properties Matcher
	}{
		{
			name: "empty_match_properties",
			properties: &regexpPropertiesMatcher{
				Services:   []*regexp.Regexp{},
				Attributes: []attributeMatcher{},
			},
		},
		{
			name: "service_name_match_regexp",
			properties: &regexpPropertiesMatcher{
				Services:   []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []attributeMatcher{},
			},
		},
		{
			name: "service_name_match_strict",
			properties: &strictPropertiesMatcher{
				Services:   []string{"svcA"},
				Attributes: []attributeMatcher{},
			},
		},
		{
			name: "span_name_match",
			properties: &regexpPropertiesMatcher{
				SpanNames:  []*regexp.Regexp{regexp.MustCompile("span.*")},
				Attributes: []attributeMatcher{},
			},
		},
		{
			name: "span_name_second_match",
			properties: &regexpPropertiesMatcher{
				SpanNames: []*regexp.Regexp{
					regexp.MustCompile("wrong.*pattern"),
					regexp.MustCompile("span.*"),
					regexp.MustCompile("yet another?pattern"),
					regexp.MustCompile("regularstring"),
				},
				Attributes: []attributeMatcher{},
			},
		},
		{
			name: "property_exact_value_match",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{},
				Attributes: []attributeMatcher{
					{
						Key:            "keyString",
						AttributeValue: newAttributeValueString("arithmetic"),
					},
					{
						Key:            "keyInt",
						AttributeValue: newAttributeValueInt(123),
					},
					{
						Key:            "keyDouble",
						AttributeValue: newAttributeValueDouble(3245.6),
					},
					{
						Key:            "keyBool",
						AttributeValue: newAttributeValueBool(true),
					},
				},
			},
		},
		{
			name: "property_exists",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []attributeMatcher{
					{
						Key:            "keyExists",
						AttributeValue: nil,
					},
				},
			},
		},
		{
			name: "match_all_settings_exists",
			properties: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []attributeMatcher{
					{
						Key:            "keyExists",
						AttributeValue: nil,
					},
					{
						Key:            "keyString",
						AttributeValue: newAttributeValueString("arithmetic"),
					},
				},
			},
		},
	}

	span := data.NewSpan()
	span.InitEmpty()
	span.SetName("spanName")
	span.Attributes().InitFromMap(map[string]data.AttributeValue{
		"keyString": data.NewAttributeValueString("arithmetic"),
		"keyInt":    data.NewAttributeValueInt(123),
		"keyDouble": data.NewAttributeValueDouble(3245.6),
		"keyBool":   data.NewAttributeValueBool(true),
		"keyExists": data.NewAttributeValueString("present"),
	})

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, tc.properties.MatchSpan(span, "svcA"))

		})
	}
}

func TestSpan_validateMatchesConfiguration(t *testing.T) {
	testcase := []struct {
		name   string
		input  MatchProperties
		output Matcher
	}{
		{
			name: "service_name_build",
			input: MatchProperties{
				Services: []string{
					"a", "b", "c",
				},
				MatchType: MatchTypeRegexp,
			},
			output: &regexpPropertiesMatcher{
				Services: []*regexp.Regexp{regexp.MustCompile("a"), regexp.MustCompile("b"), regexp.MustCompile("c")},
			},
		},

		{
			name: "attributes_build",
			input: MatchProperties{
				MatchType: MatchTypeStrict,
				Attributes: []Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &strictPropertiesMatcher{
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
			input: MatchProperties{
				MatchType: MatchTypeStrict,
				Services: []string{
					"a", "b", "c",
				},
				Attributes: []Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &strictPropertiesMatcher{
				Services: []string{
					"a", "b", "c",
				},
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
			name: "regexp_span_names",
			input: MatchProperties{
				MatchType: MatchTypeRegexp,
				SpanNames: []string{"auth.*"},
			},
			output: &regexpPropertiesMatcher{
				SpanNames: []*regexp.Regexp{regexp.MustCompile("auth.*")},
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

func newAttributeValueString(v string) *data.AttributeValue {
	attr := data.NewAttributeValueString(v)
	return &attr
}

func newAttributeValueInt(v int64) *data.AttributeValue {
	attr := data.NewAttributeValueInt(v)
	return &attr
}

func newAttributeValueDouble(v float64) *data.AttributeValue {
	attr := data.NewAttributeValueDouble(v)
	return &attr
}

func newAttributeValueBool(v bool) *data.AttributeValue {
	attr := data.NewAttributeValueBool(v)
	return &attr
}
