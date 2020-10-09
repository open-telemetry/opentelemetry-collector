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

package filterlog

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermatcher"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a log record against a
// configuration of a match.
type Matcher interface {
	MatchLogRecord(lr pdata.LogRecord, resource pdata.Resource, library pdata.InstrumentationLibrary) bool
}

// propertiesMatcher allows matching a log record against various log record properties.
type propertiesMatcher struct {
	filtermatcher.PropertiesMatcher

	// log names to compare to.
	nameFilters filterset.FilterSet
}

// NewMatcher creates a LogRecord Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForLogs(); err != nil {
		return nil, err
	}

	rm, err := filtermatcher.NewMatcher(mp)
	if err != nil {
		return nil, err
	}

	var nameFS filterset.FilterSet = nil
	if len(mp.LogNames) > 0 {
		nameFS, err = filterset.CreateFilterSet(mp.LogNames, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating log record name filters: %v", err)
		}
	}

	return &propertiesMatcher{
		PropertiesMatcher: rm,
		nameFilters:       nameFS,
	}, nil
}

// MatchLogRecord matches a log record to a set of properties.
// There are 3 sets of properties to match against.
// The log record names are matched, if specified.
// The attributes are then checked, if specified.
// At least one of log record names or attributes must be specified. It is
// supported to have more than one of these specified, and all specified must
// evaluate to true for a match to occur.
func (mp *propertiesMatcher) MatchLogRecord(lr pdata.LogRecord, resource pdata.Resource, library pdata.InstrumentationLibrary) bool {
	if mp.nameFilters != nil && !mp.nameFilters.Matches(lr.Name()) {
		return false
	}

	return mp.PropertiesMatcher.Match(lr.Attributes(), resource, library)
}
