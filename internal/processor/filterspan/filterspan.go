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
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterhelper"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/translator/conventions"
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a span against a configuration
// of a match.
type Matcher interface {
	MatchSpan(span pdata.Span, resource pdata.Resource, library pdata.InstrumentationLibrary) bool
}

// propertiesMatcher allows matching a span against various span properties.
type propertiesMatcher struct {
	// Service names to compare to.
	serviceFilters filterset.FilterSet

	// Span names to compare to.
	nameFilters filterset.FilterSet

	// Instrumentation libraries to compare against
	Libraries []instrumentationLibraryMatcher

	// The attribute values are stored in the internal format.
	Attributes filterhelper.AttributesMatcher

	Resources filterhelper.AttributesMatcher
}

type instrumentationLibraryMatcher struct {
	Name    filterset.FilterSet
	Version filterset.FilterSet
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForSpans(); err != nil {
		return nil, err
	}

	var lm []instrumentationLibraryMatcher
	for _, library := range mp.Libraries {
		name, err := filterset.CreateFilterSet([]string{library.Name}, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating library name filters: %v", err)
		}

		var version filterset.FilterSet
		if library.Version != nil {
			filter, err := filterset.CreateFilterSet([]string{*library.Version}, &mp.Config)
			if err != nil {
				return nil, fmt.Errorf("error creating library version filters: %v", err)
			}
			version = filter
		}

		lm = append(lm, instrumentationLibraryMatcher{Name: name, Version: version})
	}

	var err error
	var am filterhelper.AttributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = filterhelper.NewAttributesMatcher(mp.Config, mp.Attributes)
		if err != nil {
			return nil, fmt.Errorf("error creating attribute filters: %v", err)
		}
	}

	var rm filterhelper.AttributesMatcher
	if len(mp.Resources) > 0 {
		rm, err = filterhelper.NewAttributesMatcher(mp.Config, mp.Resources)
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
		Libraries:      lm,
		Attributes:     am,
		Resources:      rm,
	}, nil
}

// SkipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func SkipSpan(include Matcher, exclude Matcher, span pdata.Span, resource pdata.Resource, library pdata.InstrumentationLibrary) bool {
	if include != nil {
		// A false returned in this case means the span should not be processed.
		if i := include.MatchSpan(span, resource, library); !i {
			return true
		}
	}

	if exclude != nil {
		// A true returned in this case means the span should not be processed.
		if e := exclude.MatchSpan(span, resource, library); e {
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
func (mp *propertiesMatcher) MatchSpan(span pdata.Span, resource pdata.Resource, library pdata.InstrumentationLibrary) bool {
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

	for _, matcher := range mp.Libraries {
		if !matcher.Name.Matches(library.Name()) {
			return false
		}
		if matcher.Version != nil && !matcher.Version.Matches(library.Version()) {
			return false
		}
	}

	attributes := span.Attributes()
	if mp.Resources != nil && !mp.Resources.Match(attributes) {
		return false
	}

	return mp.Attributes.Match(attributes)
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
