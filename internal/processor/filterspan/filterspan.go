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
	"strconv"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterhelper"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/translator/conventions"
)

var (
	// TODO Add processor type invoking the NewMatcher in error text.
	errAtLeastOneMatchFieldNeeded = errors.New(
		`error creating processor. At least one ` +
			`of "services", "span_names" or "attributes" field must be specified"`)

	errUnexpectedAttributeType = errors.New("unexpected attribute type")
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a span against a configuration
// of a match.
type Matcher interface {
	MatchSpan(span pdata.Span, resource pdata.Resource) bool
}

// propertiesMatcher allows matching a span against various span properties.
type propertiesMatcher struct {
	// Service names to compare to.
	serviceFilters filterset.FilterSet

	// Span names to compare to.
	nameFilters filterset.FilterSet

	// The attribute values are stored in the internal format.
	Attributes attributesMatcher

	Resources attributesMatcher
}

type attributesMatcher []attributeMatcher

// attributeMatcher is a attribute key/value pair to match to.
type attributeMatcher struct {
	Key string
	// If both AttributeValue and StringFilter are nil only check for key existence.
	AttributeValue *pdata.AttributeValue
	StringFilter   filterset.FilterSet
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if len(mp.Services) == 0 && len(mp.SpanNames) == 0 && len(mp.Attributes) == 0 && len(mp.Resources) == 0 {
		return nil, errAtLeastOneMatchFieldNeeded
	}

	var err error

	var am attributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = newAttributesMatcher(mp.Config, mp.Attributes)
		if err != nil {
			return nil, fmt.Errorf("error creating attribute filters: %v", err)
		}
	}

	var rm attributesMatcher
	if len(mp.Resources) > 0 {
		rm, err = newAttributesMatcher(mp.Config, mp.Resources)
		if err != nil {
			return nil, fmt.Errorf("error creating resource filters: %v", err)
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
		Resources:      rm,
	}, nil
}

func newAttributesMatcher(config filterset.Config, attributes []Attribute) (attributesMatcher, error) {
	// Convert attribute values from mp representation to in-memory representation.
	var rawAttributes []attributeMatcher
	for _, attribute := range attributes {

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

			if config.MatchType == filterset.Regexp {
				if val.Type() != pdata.AttributeValueSTRING {
					return nil, fmt.Errorf(
						"%s=%s for %q only supports STRING, but found %s",
						filterset.MatchTypeFieldName, filterset.Regexp, attribute.Key, val.Type(),
					)
				}

				filter, err := filterset.CreateFilterSet([]string{val.StringVal()}, &config)
				if err != nil {
					return nil, err
				}
				entry.StringFilter = filter
			} else {
				entry.AttributeValue = &val
			}
		}

		rawAttributes = append(rawAttributes, entry)
	}
	return rawAttributes, nil
}

// SkipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func SkipSpan(include Matcher, exclude Matcher, span pdata.Span, resource pdata.Resource) bool {
	if include != nil {
		// A false returned in this case means the span should not be processed.
		if i := include.MatchSpan(span, resource); !i {
			return true
		}
	}

	if exclude != nil {
		// A true returned in this case means the span should not be processed.
		if e := exclude.MatchSpan(span, resource); e {
			return true
		}
	}

	return false
}

// MatchSpan matches a span and service to a set of properties.
// There are 3 sets of properties to match against.
// The service name is checked first, if specified. Then span names are matched, if specified.
// The attributes are checked last, if specified.
// At least one of services, span names or attributes must be specified. It is supported
// to have more than one of these specified, and all specified must evaluate
// to true for a match to occur.
func (mp *propertiesMatcher) MatchSpan(span pdata.Span, resource pdata.Resource) bool {
	// If a set of properties was not in the mp, all spans are considered to match on that property
	if mp.serviceFilters != nil {
		serviceName := serviceNameForResource(resource)
		if !mp.serviceFilters.Matches(serviceName) {
			return false
		}
	}

	if mp.nameFilters != nil && !mp.nameFilters.Matches(span.Name()) {
		return false
	}

	if mp.Resources != nil && !mp.Resources.match(span) {
		return false
	}

	return mp.Attributes.match(span)
}

// serviceNameForResource gets the service name for a specified Resource.
func serviceNameForResource(resource pdata.Resource) string {
	if resource.IsNil() {
		return "<nil-resource>"
	}

	service, found := resource.Attributes().Get(conventions.AttributeServiceName)
	if !found {
		return "<nil-service-name>"
	}

	return service.StringVal()
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

		if property.StringFilter != nil {
			value, err := attributeStringValue(attr)
			if err != nil || !property.StringFilter.Matches(value) {
				return false
			}
		} else if property.AttributeValue != nil {
			if !attr.Equal(*property.AttributeValue) {
				return false
			}
		}
	}
	return true
}

func attributeStringValue(attr pdata.AttributeValue) (string, error) {
	switch attr.Type() {
	case pdata.AttributeValueSTRING:
		return attr.StringVal(), nil
	case pdata.AttributeValueBOOL:
		return strconv.FormatBool(attr.BoolVal()), nil
	case pdata.AttributeValueDOUBLE:
		return strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64), nil
	case pdata.AttributeValueINT:
		return strconv.FormatInt(attr.IntVal(), 10), nil
	default:
		return "", errUnexpectedAttributeType
	}
}
