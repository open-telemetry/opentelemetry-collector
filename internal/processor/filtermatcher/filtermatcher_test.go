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

package filtermatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/translator/conventions"
)

func createConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}

func Test_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
	version := "["
	testcases := []struct {
		name        string
		property    filterconfig.MatchProperties
		errorString string
	}{
		{
			name: "regexp_match_type_for_int_attribute",
			property: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				Attributes: []filterconfig.Attribute{
					{Key: "key", Value: 1},
				},
			},
			errorString: `error creating attribute filters: match_type=regexp for "key" only supports STRING, but found INT`,
		},
		{
			name: "unknown_attribute_value",
			property: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{Key: "key", Value: []string{}},
				},
			},
			errorString: `error creating attribute filters: error unsupported value type "[]string"`,
		},
		{
			name: "invalid_regexp_pattern_attribute",
			property: filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Attributes: []filterconfig.Attribute{{Key: "key", Value: "["}},
			},
			errorString: "error creating attribute filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern_resource",
			property: filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				Resources: []filterconfig.Attribute{{Key: "key", Value: "["}},
			},
			errorString: "error creating resource filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern_library_name",
			property: filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				Libraries: []filterconfig.InstrumentationLibrary{{Name: "["}},
			},
			errorString: "error creating library name filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern_library_version",
			property: filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				Libraries: []filterconfig.InstrumentationLibrary{{Name: "lib", Version: &version}},
			},
			errorString: "error creating library version filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "empty_key_name_in_attributes_list",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"a"},
				Attributes: []filterconfig.Attribute{
					{
						Key: "",
					},
				},
			},
			errorString: "error creating attribute filters: can't have empty key in the list of attributes",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.property)
			assert.Zero(t, output)
			assert.EqualError(t, err, tc.errorString)
		})
	}
}

func Test_Matching_False(t *testing.T) {
	version := "wrong"
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "wrong_library_name",
			properties: &filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Strict),
				Services:  []string{},
				Libraries: []filterconfig.InstrumentationLibrary{{Name: "wrong"}},
			},
		},
		{
			name: "wrong_library_version",
			properties: &filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Strict),
				Services:  []string{},
				Libraries: []filterconfig.InstrumentationLibrary{{Name: "lib", Version: &version}},
			},
		},

		{
			name: "wrong_attribute_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: 1234,
					},
				},
			},
		},
		{
			name: "wrong_resource_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Resources: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: 1234,
					},
				},
			},
		},
		{
			name: "incompatible_attribute_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: "123",
					},
				},
			},
		},
		{
			name: "unsupported_attribute_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyMap",
						Value: "123",
					},
				},
			},
		},
		{
			name: "property_key_does_not_exist",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "doesnotexist",
						Value: nil,
					},
				},
			},
		},
	}

	atts := pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		"keyInt": pdata.NewAttributeValueInt(123),
		"keyMap": pdata.NewAttributeValueMap(),
	})

	library := pdata.NewInstrumentationLibrary()
	library.SetName("lib")
	library.SetVersion("ver")

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewMatcher(tc.properties)
			require.NoError(t, err)
			assert.NotNil(t, matcher)

			assert.False(t, matcher.Match(atts, resource("wrongSvc"), library))
		})
	}
}

func Test_MatchingCornerCases(t *testing.T) {
	cfg := &filterconfig.MatchProperties{
		Config: *createConfig(filterset.Strict),
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

	assert.False(t, mp.Match(pdata.NewAttributeMap(), resource("svcA"), pdata.NewInstrumentationLibrary()))
}

func Test_Matching_True(t *testing.T) {
	ver := "v.*"

	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "library_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Libraries:  []filterconfig.InstrumentationLibrary{{Name: "li.*"}},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "library_match_with_version",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Libraries:  []filterconfig.InstrumentationLibrary{{Name: "li.*", Version: &ver}},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "attribute_exact_value_match",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
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
			name: "attribute_regex_value_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyString",
						Value: "arith.*",
					},
					{
						Key:   "keyInt",
						Value: "12.*",
					},
					{
						Key:   "keyDouble",
						Value: "324.*",
					},
					{
						Key:   "keyBool",
						Value: "tr.*",
					},
				},
			},
		},
		{
			name: "resource_exact_value_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Resources: []filterconfig.Attribute{
					{
						Key:   "resString",
						Value: "arithmetic",
					},
				},
			},
		},
		{
			name: "property_exists",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA"},
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
				Services: []string{"svcA"},
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

	atts := pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		"keyString": pdata.NewAttributeValueString("arithmetic"),
		"keyInt":    pdata.NewAttributeValueInt(123),
		"keyDouble": pdata.NewAttributeValueDouble(3245.6),
		"keyBool":   pdata.NewAttributeValueBool(true),
		"keyExists": pdata.NewAttributeValueString("present"),
	})

	resource := pdata.NewResource()
	resource.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.AttributeServiceName: pdata.NewAttributeValueString("svcA"),
		"resString":                      pdata.NewAttributeValueString("arithmetic"),
	})

	library := pdata.NewInstrumentationLibrary()
	library.SetName("lib")
	library.SetVersion("ver")

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mp, err := NewMatcher(tc.properties)
			require.NoError(t, err)
			assert.NotNil(t, mp)

			assert.True(t, mp.Match(atts, resource, library))
		})
	}
}

func resource(service string) pdata.Resource {
	r := pdata.NewResource()
	r.Attributes().InitFromMap(map[string]pdata.AttributeValue{conventions.AttributeServiceName: pdata.NewAttributeValueString(service)})
	return r
}
