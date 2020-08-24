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

package filterspan

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterhelper"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

var (
	// TODO Add processor type invoking the NewMatcher in error text.
	errAtLeastOneMatchFieldNeeded = errors.New(
		`error creating processor. At least one ` +
			`of "services", "span_names" or "attributes" field must be specified"`)
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a span against a configuration
// of a match.
type Matcher interface {
	MatchSpan(span pdata.Span, serviceName string) bool
}

// propertiesMatcher allows matching a span against various span properties.
type propertiesMatcher struct {
	// Service names to compare to.
	serviceFilters filterset.FilterSet

	// Span names to compare to.
	nameFilters filterset.FilterSet

	// The attribute values are stored in the internal format.
	Attributes attributesMatcher
}

type attributesMatcher []attributeMatcher

// attributeMatcher is a attribute key/value pair to match to.
type attributeMatcher struct {
	Key string
	// If nil only check for key existence.
	AttributeValue *pdata.AttributeValue
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if len(mp.Services) == 0 && len(mp.SpanNames) == 0 && len(mp.Attributes) == 0 {
		return nil, errAtLeastOneMatchFieldNeeded
	}

	var err error

	var am attributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = newAttributesMatcher(mp)
		if err != nil {
			return nil, err
		}
	}

	var serviceFS filterset.FilterSet = nil
	if len(mp.Services) > 0 {
		serviceFS, err = filterset.CreateFilterSet(mp.Services, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating service name filters: %v", err)
		}
	}

	var nameFS filterset.FilterSet = nil
	if len(mp.SpanNames) > 0 {
		nameFS, err = filterset.CreateFilterSet(mp.SpanNames, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating span name filters: %v", err)
		}
	}

	return &propertiesMatcher{
		serviceFilters: serviceFS,
		nameFilters:    nameFS,
		Attributes:     am,
	}, nil
}

func newAttributesMatcher(mp *MatchProperties) (attributesMatcher, error) {
	// attribute matching is only supported with strict matching
	if mp.Config.MatchType != filterset.Strict {
		return nil, fmt.Errorf(
			"%s=%s is not supported for %q",
			filterset.MatchTypeFieldName, filterset.Regexp, AttributesFieldName,
		)
	}

	// Convert attribute values from mp representation to in-memory representation.
	var rawAttributes []attributeMatcher
	for _, attribute := range mp.Attributes {

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
func (mp *propertiesMatcher) MatchSpan(span pdata.Span, serviceName string) bool {
	// If a set of properties was not in the mp, all spans are considered to match on that property
	if mp.serviceFilters != nil && !mp.serviceFilters.Matches(serviceName) {
		return false
	}

	if mp.nameFilters != nil && !mp.nameFilters.Matches(span.Name()) {
		return false
	}

	// Service name and span name matched. Now match attributes.
	return mp.Attributes.match(span)
}

// match attributes specification against a span.
func (ma attributesMatcher) match(span pdata.Span) bool {
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
