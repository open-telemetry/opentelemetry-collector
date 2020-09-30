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
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterhelper"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

// TODO: Modify Matcher to invoke both the include and exclude properties so
// calling processors will always have the same logic.
// Matcher is an interface that allows matching a log record against a
// configuration of a match.
type Matcher interface {
	MatchLogRecord(lr pdata.LogRecord) bool
}

// propertiesMatcher allows matching a log record against various log record properties.
type propertiesMatcher struct {
	// log names to compare to.
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

// NewMatcher creates a LogRecord Matcher that matches based on the given MatchProperties.
func NewMatcher(mp *filterconfig.MatchProperties) (Matcher, error) {
	if mp == nil {
		return nil, nil
	}

	if err := mp.ValidateForLogs(); err != nil {
		return nil, err
	}

	var err error

	var am attributesMatcher
	if len(mp.Attributes) > 0 {
		am, err = newAttributesMatcher(mp)
		if err != nil {
			return nil, err
		}
	}

	var nameFS filterset.FilterSet = nil
	if len(mp.LogNames) > 0 {
		nameFS, err = filterset.CreateFilterSet(mp.LogNames, &mp.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating log record name filters: %v", err)
		}
	}

	return &propertiesMatcher{
		nameFilters: nameFS,
		Attributes:  am,
	}, nil
}

func newAttributesMatcher(mp *filterconfig.MatchProperties) (attributesMatcher, error) {
	// attribute matching is only supported with strict matching
	if mp.Config.MatchType != filterset.Strict {
		return nil, fmt.Errorf(
			"%s=%s is not supported for %q",
			filterset.MatchTypeFieldName, filterset.Regexp, filterconfig.AttributesFieldName,
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

// MatchLogRecord matches a log record to a set of properties.
// There are 3 sets of properties to match against.
// The log record names are matched, if specified.
// The attributes are then checked, if specified.
// At least one of log record names or attributes must be specified. It is
// supported to have more than one of these specified, and all specified must
// evaluate to true for a match to occur.
func (mp *propertiesMatcher) MatchLogRecord(lr pdata.LogRecord) bool {
	if mp.nameFilters != nil && !mp.nameFilters.Matches(lr.Name()) {
		return false
	}

	return mp.Attributes.match(lr)
}

// match attributes specification against a log record.
func (ma attributesMatcher) match(lr pdata.LogRecord) bool {
	// If there are no attributes to match against, the log matches.
	if len(ma) == 0 {
		return true
	}

	attrs := lr.Attributes()
	// At this point, it is expected of the log record to have attributes
	// because of len(ma) != 0. This means for log records with no attributes,
	// it does not match.
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
