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
	"go.opentelemetry.io/collector/internal/processor/filtermatcher"
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
	filtermatcher.PropertiesMatcher

	// Service names to compare to.
	serviceFilters filterset.FilterSet

	// Span names to compare to.
	nameFilters filterset.FilterSet
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForSpans(); err != nil {
		return nil, err
	}

	rm, err := filtermatcher.NewMatcher(mp)
	if err != nil {
		return nil, err
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
		PropertiesMatcher: rm,
		serviceFilters:    serviceFS,
		nameFilters:       nameFS,
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
// see filterconfig.MatchProperties for more details
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

	return mp.PropertiesMatcher.Match(span.Attributes(), resource, library)
}

// serviceNameForResource gets the service name for a specified Resource.
func serviceNameForResource(resource pdata.Resource) string {
	service, found := resource.Attributes().Get(conventions.AttributeServiceName)
	if !found {
		return "<nil-service-name>"
	}

	return service.StringVal()
}
