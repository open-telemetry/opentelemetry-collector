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
	"errors"
	"fmt"
	"regexp"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/cast"
)

var (
	// TODO Add processor type invoking the BuildMatchProperties in error text.
	errAtLeastOneMatchFieldNeeded = errors.New(
		`error creating processor. At least one ` +
			`of "services", "span_names" or "attributes" field must be specified"`)

	errInvalidMatchType = fmt.Errorf(
		`match_type must be either %q or %q`, MatchTypeStrict, MatchTypeRegexp)
)

// matchingProperties is an interface that allows matching a span against a configuration
// of a match.
type MatchingProperties interface {
	MatchSpan(span *tracepb.Span, serviceName string) bool
}

type matchAttributes []matchAttribute

// strictMatchingProperties allows matching a span against a "strict" match type
// configuration.
type strictMatchingProperties struct {
	// Service names to compare to.
	Services []string

	// Span names to compare to.
	SpanNames []string

	// The attribute values are stored in the internal format.
	Attributes matchAttributes
}

// strictMatchingProperties allows matching a span against a "regexp" match type
// configuration.
type regexpMatchingProperties struct {
	// Precompiled service name regexp-es.
	Services []*regexp.Regexp

	// Precompiled span name regexp-es.
	SpanNames []*regexp.Regexp

	// The attribute values are stored in the internal format.
	Attributes matchAttributes
}

// matchAttribute is a attribute key/value pair to match to.
type matchAttribute struct {
	Key            string
	AttributeValue *tracepb.AttributeValue
}

func BuildMatchProperties(config *MatchProperties) (MatchingProperties, error) {
	if config == nil {
		return nil, nil
	}

	if len(config.Services) == 0 && len(config.SpanNames) == 0 && len(config.Attributes) == 0 {
		return nil, errAtLeastOneMatchFieldNeeded
	}

	var properties MatchingProperties
	var err error
	switch config.MatchType {
	case MatchTypeStrict:
		properties, err = buildStrictMatchingProperties(config)
	case MatchTypeRegexp:
		properties, err = buildRegexpMatchingProperties(config)
	default:
		return nil, errInvalidMatchType
	}
	if err != nil {
		return nil, err
	}

	return properties, nil
}

func buildStrictMatchingProperties(config *MatchProperties) (*strictMatchingProperties, error) {
	properties := &strictMatchingProperties{
		Services:  config.Services,
		SpanNames: config.SpanNames,
	}

	var err error
	properties.Attributes, err = buildAttributesMatches(config)
	if err != nil {
		return nil, err
	}

	return properties, nil
}

func buildRegexpMatchingProperties(config *MatchProperties) (*regexpMatchingProperties, error) {
	properties := &regexpMatchingProperties{}

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

func buildAttributesMatches(config *MatchProperties) (matchAttributes, error) {
	// Convert attribute values from config representation to in-memory representation.
	var rawAttributes []matchAttribute
	for _, attribute := range config.Attributes {

		if attribute.Key == "" {
			return nil, errors.New("error creating processor. Can't have empty key in the list of attributes")
		}

		entry := matchAttribute{
			Key: attribute.Key,
		}
		if attribute.Value != nil {
			val, err := AttributeValue(attribute.Value)
			if err != nil {
				return nil, err
			}
			entry.AttributeValue = val
		}

		rawAttributes = append(rawAttributes, entry)
	}
	return rawAttributes, nil
}

// AttributeValue is used to convert the raw `value` from ActionKeyValue to the supported trace attribute values.
func AttributeValue(value interface{}) (*tracepb.AttributeValue, error) {
	attrib := &tracepb.AttributeValue{}
	switch val := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		attrib.Value = &tracepb.AttributeValue_IntValue{IntValue: cast.ToInt64(val)}
	case float32, float64:
		attrib.Value = &tracepb.AttributeValue_DoubleValue{DoubleValue: cast.ToFloat64(val)}
	case string:
		attrib.Value = &tracepb.AttributeValue_StringValue{
			StringValue: &tracepb.TruncatableString{Value: val},
		}
	case bool:
		attrib.Value = &tracepb.AttributeValue_BoolValue{BoolValue: val}
	default:
		return nil, fmt.Errorf("error unsupported value type \"%T\"", value)
	}
	return attrib, nil
}

// MatchSpan matches a span and service to a set of properties.
// There are 3 sets of properties to match against.
// The service name is checked first, if specified. Then span names are matched, if specified.
// The attributes are checked last, if specified.
// At least one of services, span names or attributes must be specified. It is supported
// to have more than one of these specified, and all specified must evaluate
// to true for a match to occur.
func (mp *strictMatchingProperties) MatchSpan(span *tracepb.Span, serviceName string) bool {

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
		var spanName string
		if span.Name != nil {
			spanName = span.Name.Value
		}
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
func (mp *regexpMatchingProperties) MatchSpan(span *tracepb.Span, serviceName string) bool {

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
		var spanName string
		if span.Name != nil {
			spanName = span.Name.Value
		}
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
func (ma matchAttributes) match(span *tracepb.Span) bool {
	// If there are no attributes to match against, the span matches.
	if len(ma) == 0 {
		return true
	}

	// At this point, it is expected of the span to have attributes because of
	// len(ma) != 0. This means for spans with no attributes, it does not match.
	if span.Attributes == nil || len(span.Attributes.AttributeMap) == 0 {
		return false
	}

	// Check that all expected properties are set.
	for _, property := range ma {
		val, exist := span.Attributes.AttributeMap[property.Key]
		if !exist {
			return false
		}

		// This is for the case of checking that the key existed.
		if property.AttributeValue == nil {
			continue
		}

		var isMatch bool
		switch attribValue := val.Value.(type) {
		case *tracepb.AttributeValue_StringValue:
			if sv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_StringValue); ok {
				isMatch = attribValue.StringValue.GetValue() == sv.StringValue.GetValue()
			}
		case *tracepb.AttributeValue_IntValue:
			if iv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_IntValue); ok {
				isMatch = attribValue.IntValue == iv.IntValue
			}
		case *tracepb.AttributeValue_BoolValue:
			if bv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_BoolValue); ok {
				isMatch = attribValue.BoolValue == bv.BoolValue
			}
		case *tracepb.AttributeValue_DoubleValue:
			if dv, ok := property.AttributeValue.GetValue().(*tracepb.AttributeValue_DoubleValue); ok {
				isMatch = attribValue.DoubleValue == dv.DoubleValue
			}
		}
		if !isMatch {
			return false
		}
	}
	return true
}
