// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filtermatcher

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

type instrumentationLibraryMatcher struct {
	Name    filterset.FilterSet
	Version filterset.FilterSet
}

// propertiesMatcher allows matching a span against various span properties.
type PropertiesMatcher struct {
	// Instrumentation libraries to compare against
	libraries []instrumentationLibraryMatcher

	// The attribute values are stored in the internal format.
	attributes attributesMatcher

	// The attribute values are stored in the internal format.
	resources attributesMatcher
}

// NewMatcher creates a span Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (PropertiesMatcher, error) {
	var lm []instrumentationLibraryMatcher
	for _, library := range mp.Libraries {
		name, err := filterset.CreateFilterSet([]string{library.Name}, &mp.Config)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating library name filters: %v", err)
		}

		var version filterset.FilterSet
		if library.Version != nil {
			filter, err := filterset.CreateFilterSet([]string{*library.Version}, &mp.Config)
			if err != nil {
				return PropertiesMatcher{}, fmt.Errorf("error creating library version filters: %v", err)
			}
			version = filter
		}

		lm = append(lm, instrumentationLibraryMatcher{Name: name, Version: version})
	}

	var err error
	var am attributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = newAttributesMatcher(mp.Config, mp.Attributes)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating attribute filters: %v", err)
		}
	}

	var rm attributesMatcher
	if len(mp.Resources) > 0 {
		rm, err = newAttributesMatcher(mp.Config, mp.Resources)
		if err != nil {
			return PropertiesMatcher{}, fmt.Errorf("error creating resource filters: %v", err)
		}
	}

	return PropertiesMatcher{
		libraries:  lm,
		attributes: am,
		resources:  rm,
	}, nil
}

// Match matches a span or log to a set of properties.
func (mp *PropertiesMatcher) Match(attributes pdata.AttributeMap, resource pdata.Resource, library pdata.InstrumentationLibrary) bool {
	for _, matcher := range mp.libraries {
		if !matcher.Name.Matches(library.Name()) {
			return false
		}
		if matcher.Version != nil && !matcher.Version.Matches(library.Version()) {
			return false
		}
	}

	if mp.resources != nil && !mp.resources.Match(resource.Attributes()) {
		return false
	}

	return mp.attributes.Match(attributes)
}
