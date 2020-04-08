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
	"errors"
	"fmt"
	"regexp"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterhelper"
)

var (
	// TODO Add processor type invoking the NewMatcher in error text.
	errAtLeastOneMatchFieldNeeded = errors.New(
		`error creating processor. At least one ` +
			`of "services", "span_names" or "attributes" field must be specified"`)

	errInvalidMatchType = fmt.Errorf(
		`match_type must be either %q or %q`, MatchTypeStrict, MatchTypeRegexp)
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a span against a configuration
// of a match.
type Matcher interface {
	MatchSpan(span data.Span, serviceName string) bool
}

type attributesMatcher []attributeMatcher

// strictPropertiesMatcher allows matching a span against a "strict" match type
// configuration.
type strictPropertiesMatcher struct {
	// Service names to compare to.
	Services []string

	// Span names to compare to.
	SpanNames []string

	// The attribute values are stored in the internal format.
	Attributes attributesMatcher
}

// regexpPropertiesMatcher allows matching a span against a "regexp" match type
// configuration.
type regexpPropertiesMatcher struct {
	// Precompiled service name regexp-es.
	Services []*regexp.Regexp

	// Precompiled span name regexp-es.
	SpanNames []*regexp.Regexp

	// The attribute values are stored in the internal format.
	Attributes attributesMatcher
}

// attributeMatcher is a attribute key/value pair to match to.
type attributeMatcher struct {
	Key string
	// If nil only check for key existence.
	AttributeValue *data.AttributeValue
}

func NewMatcher(config *MatchProperties) (Matcher, error) {
	if config == nil {
		return nil, nil
	}

	if len(config.Services) == 0 && len(config.SpanNames) == 0 && len(config.Attributes) == 0 {
		return nil, errAtLeastOneMatchFieldNeeded
	}

	var properties Matcher
	var err error
	switch config.MatchType {
	case MatchTypeStrict:
		properties, err = newStrictPropertiesMatcher(config)
	case MatchTypeRegexp:
		properties, err = newRegexpPropertiesMatcher(config)
	default:
		return nil, errInvalidMatchType
	}
	if err != nil {
		return nil, err
	}

	return properties, nil
}

func newStrictPropertiesMatcher(config *MatchProperties) (*strictPropertiesMatcher, error) {
	properties := &strictPropertiesMatcher{
		Services:  config.Services,
		SpanNames: config.SpanNames,
	}

	var err error
	properties.Attributes, err = newAttributesMatcher(config)
	if err != nil {
		return nil, err
	}

	return properties, nil
}

func newRegexpPropertiesMatcher(config *MatchProperties) (*regexpPropertiesMatcher, error) {
	properties := &regexpPropertiesMatcher{}

	// Precompile Services regexp patterns.
	for _, pattern := range config.Services {
		g, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf(
				"error creating processor. %s is not a valid service name regexp pattern",
				pattern,
			)
		}
		properties.Services = append(properties.Services, g)
	}

	// Precompile SpanNames regexp patterns.
	for _, pattern := range config.SpanNames {
		g, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf(
				"error creating processor. %s is not a valid span name regexp pattern",
				pattern,
			)
		}
		properties.SpanNames = append(properties.SpanNames, g)
	}

	if len(config.Attributes) > 0 {
		return nil, fmt.Errorf(
			"%s=%s is not supported for %q",
			MatchTypeFieldName, MatchTypeRegexp, AttributesFieldName,
		)
	}

	return properties, nil
}

func newAttributesMatcher(config *MatchProperties) (attributesMatcher, error) {
	// Convert attribute values from config representation to in-memory representation.
	var rawAttributes []attributeMatcher
	for _, attribute := range config.Attributes {

		if attribute.Key == "" {
			return nil, errors.New("error creating processor. Can't have empty key in the list of attributes")
		}

		entry := attributeMatcher{
			Key: attribute.Key,
		}
		if attribute.Value != nil {
			val, err := filterhelper.NewAttributeValueRaw(attribute.Value)
			if err != nil {
				return nil, err
			}
			entry.AttributeValue = &val
		}

		rawAttributes = append(rawAttributes, entry)
	}
	return rawAttributes, nil
}

// MatchSpan matches a span and service to a set of properties.
// There are 3 sets of properties to match against.
// The service name is checked first, if specified. Then span names are matched, if specified.
// The attributes are checked last, if specified.
// At least one of services, span names or attributes must be specified. It is supported
// to have more than one of these specified, and all specified must evaluate
// to true for a match to occur.
func (mp *strictPropertiesMatcher) MatchSpan(span data.Span, serviceName string) bool {
	if len(mp.Services) > 0 {
		// Verify service name matches at least one of the items.
		matched := false
		for _, item := range mp.Services {
			if item == serviceName {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(mp.SpanNames) > 0 {
		// SpanNames condition is specified. Check if span name matches the condition.
		spanName := span.Name()

		// Verify span name matches at least one of the items.
		matched := false
		for _, item := range mp.SpanNames {
			if item == spanName {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Service name and span name matched. Now match attributes.
	return mp.Attributes.match(span)
}

// MatchSpan matches a span and service to a set of properties.
// There are 3 sets of properties to match against.
// The service name is checked first, if specified. Then span names are matched, if specified.
// The attributes are checked last, if specified.
// At least one of services, span names or attributes must be specified. It is supported
// to have more than one of these specified, and all specified must evaluate
// to true for a match to occur.
func (mp *regexpPropertiesMatcher) MatchSpan(span data.Span, serviceName string) bool {

	if len(mp.Services) > 0 {
		// Verify service name matches at least one of the regexp patterns.
		matched := false
		for _, re := range mp.Services {
			if re.MatchString(serviceName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(mp.SpanNames) > 0 {
		// SpanNames condition is specified. Check if span name matches the condition.
		spanName := span.Name()

		// Verify span name matches at least one of the regexp patterns.
		matched := false
		for _, re := range mp.SpanNames {
			if re.MatchString(spanName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Service name and span name matched. Now match attributes.
	return mp.Attributes.match(span)
}

// match attributes specification against a span.
func (ma attributesMatcher) match(span data.Span) bool {
	// If there are no attributes to match against, the span matches.
	if len(ma) == 0 {
		return true
	}

	attrs := span.Attributes()
	// At this point, it is expected of the span to have attributes because of
	// len(ma) != 0. This means for spans with no attributes, it does not match.
	if attrs.Len() == 0 {
		return false
	}

	// Check that all expected properties are set.
	for _, property := range ma {
		attr, exist := attrs.Get(property.Key)
		if !exist {
			return false
		}

		// This is for the case of checking that the key existed.
		if property.AttributeValue == nil {
			continue
		}

		if !attr.Equal(*property.AttributeValue) {
			return false
		}
	}
	return true
}
