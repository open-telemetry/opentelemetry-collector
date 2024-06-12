// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"errors"
	"fmt"
	"strings"
)

type PipelineID struct {
	typeVal DataType `mapstructure:"-"`
	nameVal string   `mapstructure:"-"`
}

// Type returns the type of the component.
func (id PipelineID) Type() DataType {
	return id.typeVal
}

// Name returns the custom name of the component.
func (id PipelineID) Name() string {
	return id.nameVal
}

// NewPipelineID returns a new PipelineID with the given DataType and empty name.
func NewPipelineID(typeVal DataType) PipelineID {
	return PipelineID{typeVal: typeVal}
}

// NewPipelineIDWithName returns a new PipelineID with the given DataType and name.
func NewPipelineIDWithName(typeVal DataType, nameVal string) PipelineID {
	return PipelineID{typeVal: typeVal, nameVal: nameVal}
}

// MarshalText implements the encoding.TextMarshaler interface.
// This marshals the type and name as one string in the config.
func (id PipelineID) MarshalText() (text []byte, err error) {
	return []byte(id.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (id *PipelineID) UnmarshalText(text []byte) error {
	idStr := string(text)
	items := strings.SplitN(idStr, TypeAndNameSeparator, 2)
	var typeStr, nameStr string
	if len(items) >= 1 {
		typeStr = strings.TrimSpace(items[0])
	}

	if len(items) == 1 && typeStr == "" {
		return errors.New("id must not be empty")
	}

	if typeStr == "" {
		return fmt.Errorf("in %q id: the part before %s should not be empty", idStr, TypeAndNameSeparator)
	}

	if len(items) > 1 {
		// "name" part is present.
		nameStr = strings.TrimSpace(items[1])
		if nameStr == "" {
			return fmt.Errorf("in %q id: the part after %s should not be empty", idStr, TypeAndNameSeparator)
		}
	}

	var err error

	if id.typeVal, err = newDataType(typeStr); err != nil {
		return fmt.Errorf("in %q id: %w", idStr, err)
	}
	id.nameVal = nameStr

	return nil
}

// String returns the ID string representation as "type[/name]" format.
func (id PipelineID) String() string {
	if id.nameVal == "" {
		return id.typeVal.String()
	}

	return id.typeVal.String() + TypeAndNameSeparator + id.nameVal
}
