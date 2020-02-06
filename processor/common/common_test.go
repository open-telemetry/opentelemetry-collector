// Copyright 2019, OpenTelemetry Authors
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

package common

import (
	"regexp"
	"testing"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommon_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
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
			output, err := BuildMatchProperties(&tc.property)
			assert.Nil(t, output)
			require.NotNil(t, err)
			assert.Equal(t, tc.errorString, err.Error())
		})
	}
}

func TestCommon_Matching_False(t *testing.T) {
	testcases := []struct {
		name       string
		properties MatchingProperties
	}{
		{
			name: "service_name_doesnt_match_regexp",
			properties: &regexpMatchingProperties{
				Services:   []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []matchAttribute{},
			},
		},

		{
			name: "service_name_doesnt_match_strict",
			properties: &strictMatchingProperties{
				Services:   []string{"svcA"},
				Attributes: []matchAttribute{},
			},
		},

		{
			name: "span_name_doesnt_match",
			properties: &regexpMatchingProperties{
				SpanNames:  []*regexp.Regexp{regexp.MustCompile("spanNo.*Name")},
				Attributes: []matchAttribute{},
			},
		},

		{
			name: "span_name_doesnt_match_any",
			properties: &regexpMatchingProperties{
				SpanNames: []*regexp.Regexp{
					regexp.MustCompile("spanNo.*Name"),
					regexp.MustCompile("non-matching?pattern"),
					regexp.MustCompile("regular string"),
				},
				Attributes: []matchAttribute{},
			},
		},

		{
			name: "wrong_property_value",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{},
				Attributes: []matchAttribute{
					{
						Key: "keyInt",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_IntValue{
								IntValue: 1234,
							},
						},
					},
				},
			},
		},
		{
			name: "incompatible_property_value",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{},
				Attributes: []matchAttribute{
					{
						Key: "keyInt",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "123"},
							},
						},
					},
				},
			},
		},
		{
			name: "property_key_does_not_exist",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{},
				Attributes: []matchAttribute{
					{
						Key:            "doesnotexist",
						AttributeValue: nil,
					},
				},
			},
		},
	}

	span := &tracepb.Span{
		Name: &tracepb.TruncatableString{Value: "spanName"},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				"keyInt": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, tc.properties.MatchSpan(span, "wrongSvc"))
		})
	}
}

func TestCommon_MatchingCornerCases(t *testing.T) {
	mp := &regexpMatchingProperties{
		Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
		Attributes: []matchAttribute{
			{
				Key:            "keyOne",
				AttributeValue: nil,
			},
		},
	}
	testcases := []struct {
		name string
		span *tracepb.Span
	}{
		{
			name: "nil_attributes",
			span: &tracepb.Span{
				Attributes: nil,
			},
		},
		{
			name: "default_attributes",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{},
			},
		},
		{
			name: "empty_map",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, mp.MatchSpan(tc.span, "svcA"))
		})
	}
}

func TestCommon_MissingServiceName(t *testing.T) {
	mp := &regexpMatchingProperties{
		Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
	}
	testcases := []struct {
		name string
		span *tracepb.Span
	}{
		{
			name: "nil_attributes",
			span: &tracepb.Span{
				Attributes: nil,
			},
		},
		{
			name: "default_attributes",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{},
			},
		},
		{
			name: "empty_map",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, mp.MatchSpan(tc.span, ""))
		})
	}
}

func TestCommon_Matching_True(t *testing.T) {
	testcases := []struct {
		name       string
		properties MatchingProperties
	}{
		{
			name: "empty_match_properties",
			properties: &regexpMatchingProperties{
				Services:   []*regexp.Regexp{},
				Attributes: []matchAttribute{},
			},
		},
		{
			name: "service_name_match_regexp",
			properties: &regexpMatchingProperties{
				Services:   []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []matchAttribute{},
			},
		},
		{
			name: "service_name_match_strict",
			properties: &strictMatchingProperties{
				Services:   []string{"svcA"},
				Attributes: []matchAttribute{},
			},
		},
		{
			name: "span_name_match",
			properties: &regexpMatchingProperties{
				SpanNames:  []*regexp.Regexp{regexp.MustCompile("span.*")},
				Attributes: []matchAttribute{},
			},
		},
		{
			name: "span_name_second_match",
			properties: &regexpMatchingProperties{
				SpanNames: []*regexp.Regexp{
					regexp.MustCompile("wrong.*pattern"),
					regexp.MustCompile("span.*"),
					regexp.MustCompile("yet another?pattern"),
					regexp.MustCompile("regularstring"),
				},
				Attributes: []matchAttribute{},
			},
		},
		{
			name: "property_exact_value_match",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{},
				Attributes: []matchAttribute{
					{
						Key: "keyString",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "arithmetic"},
							},
						},
					},
					{
						Key: "keyInt",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_IntValue{
								IntValue: 123,
							},
						},
					},
					{
						Key: "keyDouble",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_DoubleValue{
								DoubleValue: cast.ToFloat64(3245.6),
							},
						},
					},
					{
						Key: "keyBool",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
						},
					},
				},
			},
		},
		{
			name: "property_exists",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []matchAttribute{
					{
						Key:            "keyExists",
						AttributeValue: nil,
					},
				},
			},
		},
		{
			name: "match_all_settings_exists",
			properties: &regexpMatchingProperties{
				Services: []*regexp.Regexp{regexp.MustCompile("svcA")},
				Attributes: []matchAttribute{
					{
						Key:            "keyExists",
						AttributeValue: nil,
					},
					{
						Key: "keyString",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "arithmetic"},
							},
						},
					},
				},
			},
		},
	}

	span := &tracepb.Span{
		Name: &tracepb.TruncatableString{Value: "spanName"},
		Attributes: &tracepb.Span_Attributes{
			AttributeMap: map[string]*tracepb.AttributeValue{
				"keyString": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "arithmetic"}},
				},
				"keyInt": {
					Value: &tracepb.AttributeValue_IntValue{IntValue: 123},
				},
				"keyDouble": {
					Value: &tracepb.AttributeValue_DoubleValue{
						DoubleValue: cast.ToFloat64(3245.6),
					},
				},
				"keyBool": {
					Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
				},
				"keyExists": {
					Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "present"}},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, tc.properties.MatchSpan(span, "svcA"))

		})
	}
}

func TestFactory_validateMatchesConfiguration(t *testing.T) {
	testcase := []struct {
		name   string
		input  MatchProperties
		output MatchingProperties
	}{
		{
			name: "service_name_build",
			input: MatchProperties{
				Services: []string{
					"a", "b", "c",
				},
				MatchType: MatchTypeRegexp,
			},
			output: &regexpMatchingProperties{
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
			output: &strictMatchingProperties{
				Attributes: []matchAttribute{
					{
						Key: "key1",
					},
					{
						Key: "key2",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(1234)},
						},
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
			output: &strictMatchingProperties{
				Services: []string{
					"a", "b", "c",
				},
				Attributes: []matchAttribute{
					{
						Key: "key1",
					},
					{
						Key: "key2",
						AttributeValue: &tracepb.AttributeValue{
							Value: &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(1234)},
						},
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
			output: &regexpMatchingProperties{
				SpanNames: []*regexp.Regexp{regexp.MustCompile("auth.*")},
			},
		},
	}
	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			output, err := BuildMatchProperties(&tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.output, output)
		})
	}
}

func TestCommon_AttributeValue(t *testing.T) {
	val, err := AttributeValue(123)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(123)}}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(234.129312)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(234.129312)}}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(true)
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_BoolValue{BoolValue: true},
	}, val)
	assert.Nil(t, err)

	val, err = AttributeValue("bob the builder")
	assert.Equal(t, &tracepb.AttributeValue{
		Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "bob the builder"}},
	}, val)
	assert.Nil(t, err)

	val, err = AttributeValue(nil)
	assert.Nil(t, val)
	assert.Equal(t, "error unsupported value type \"<nil>\"", err.Error())

	val, err = AttributeValue(MatchProperties{})
	assert.Nil(t, val)
	assert.Equal(t, "error unsupported value type \"common.MatchProperties\"", err.Error())
}
