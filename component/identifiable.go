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

package component // import "go.opentelemetry.io/collector/component"

import (
	"errors"
	"fmt"
	"strings"
)

// typeAndNameSeparator is the separator that is used between type and name in type/name composite keys.
const typeAndNameSeparator = "/"

// ID represents the identity for a component. It combines two values:
// * type - the Type of the component.
// * name - the name of that component.
// The component ID (combination type + name) is unique for a given component.Kind.
type ID struct {
	typeVal Type   `mapstructure:"-"`
	nameVal string `mapstructure:"-"`
}

// NewID returns a new ID with the given Type and empty name.
func NewID(typeVal Type) ID {
	return ID{typeVal: typeVal}
}

// NewIDWithName returns a new ID with the given Type and name.
func NewIDWithName(typeVal Type, nameVal string) ID {
	return ID{typeVal: typeVal, nameVal: nameVal}
}

// Type returns the type of the component.
func (id ID) Type() Type {
	return id.typeVal
}

// Name returns the custom name of the component.
func (id ID) Name() string {
	return id.nameVal
}

// MarshalText implements the encoding.TextMarshaler interface.
// This marshals the type and name as one string in the config.
func (id ID) MarshalText() (text []byte, err error) {
	return []byte(id.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (id *ID) UnmarshalText(text []byte) error {
	idStr := string(text)
	items := strings.SplitN(idStr, typeAndNameSeparator, 2)
	if len(items) >= 1 {
		id.typeVal = Type(strings.TrimSpace(items[0]))
	}

	if len(items) == 1 && id.typeVal == "" {
		return errors.New("id must not be empty")
	}

	if id.typeVal == "" {
		return fmt.Errorf("in %q id: the part before %s should not be empty", idStr, typeAndNameSeparator)
	}

	if len(items) > 1 {
		// "name" part is present.
		id.nameVal = strings.TrimSpace(items[1])
		if id.nameVal == "" {
			return fmt.Errorf("in %q id: the part after %s should not be empty", idStr, typeAndNameSeparator)
		}
	}

	return nil
}

// String returns the ID string representation as "type[/name]" format.
func (id ID) String() string {
	if id.nameVal == "" {
		return string(id.typeVal)
	}

	return string(id.typeVal) + typeAndNameSeparator + id.nameVal
}
