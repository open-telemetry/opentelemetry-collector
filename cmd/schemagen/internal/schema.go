// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/schemagen/internal"

import (
	"bytes"
	"encoding/json"
	"errors"

	"go.yaml.in/yaml/v3"
)

type SchemaElement interface {
	setIsPointer(value bool)
	setDescription(description string)
	setOptional(value bool)
}

type SchemaObject interface {
	AddProperty(name string, property SchemaElement)
	AddEmbedded(element SchemaElement)
}

type BaseSchemaElement struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	IsPointer   bool   `json:"x-pointer,omitempty" yaml:"x-pointer,omitempty"`
	IsOptional  bool   `json:"x-optional,omitempty" yaml:"x-optional,omitempty"`
}

func (b *BaseSchemaElement) setIsPointer(value bool) {
	b.IsPointer = value
}

func (b *BaseSchemaElement) setDescription(value string) {
	b.Description = value
}

func (b *BaseSchemaElement) setOptional(value bool) {
	b.IsOptional = value
}

type RefSchemaElement struct {
	BaseSchemaElement `json:",inline" yaml:",inline"`
	Ref               string `json:"$ref" yaml:"$ref"`
}

type FieldSchemaElement struct {
	BaseSchemaElement `json:",inline" yaml:",inline"`
	ElementType       SchemaType `json:"type,omitempty" yaml:"type,omitempty"`
	CustomElementType string     `json:"x-customType,omitempty" yaml:"x-customType,omitempty"`
	Format            string     `json:"format,omitempty" yaml:"format,omitempty"`
}

type ArraySchemaElement struct {
	FieldSchemaElement `json:",inline" yaml:",inline"`
	Items              SchemaElement `json:"items" yaml:"items"`
}
type ObjectSchemaElement struct {
	SchemaObject         `json:"-" yaml:"-"`
	FieldSchemaElement   `json:",inline" yaml:",inline"`
	Properties           map[string]SchemaElement `json:"properties,omitempty" yaml:"properties,omitempty"`
	AdditionalProperties SchemaElement            `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	AllOf                []SchemaElement          `json:"allOf,omitempty" yaml:"allOf,omitempty"`
}

func (s *ObjectSchemaElement) AddProperty(name string, property SchemaElement) {
	if s.Properties == nil {
		s.Properties = make(map[string]SchemaElement)
	}
	s.Properties[name] = property
}

func (s *ObjectSchemaElement) AddEmbedded(element SchemaElement) {
	// prevent duplicates
	if re, ok := element.(*RefSchemaElement); ok {
		ref := re.Ref
		for _, refEl := range s.AllOf {
			if r, ok := refEl.(*RefSchemaElement); ok && r.Ref == ref {
				return
			}
		}
	}
	s.AllOf = append(s.AllOf, element)
}

type DefsSchemaElement map[string]SchemaElement

func (d DefsSchemaElement) AddDef(name string, property SchemaElement) {
	d[name] = property
}

type Schema struct {
	Defs                DefsSchemaElement `json:"$defs,omitempty" yaml:"$defs,omitempty"`
	ObjectSchemaElement `json:",inline" yaml:",inline"`
}

func (s *Schema) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

func (s *Schema) ToYAML() ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)

	enc.SetIndent(2)

	if err := enc.Encode(s); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func CreateSchema() *Schema {
	return &Schema{
		Defs: DefsSchemaElement{},
	}
}

func CreateSimpleField(fieldType SchemaType, description string) *FieldSchemaElement {
	return &FieldSchemaElement{
		BaseSchemaElement: BaseSchemaElement{
			Description: description,
		},
		ElementType: fieldType,
	}
}

func CreateArrayField(itemType SchemaElement, description string) *ArraySchemaElement {
	return &ArraySchemaElement{
		FieldSchemaElement: FieldSchemaElement{
			BaseSchemaElement: BaseSchemaElement{
				Description: description,
			},
			ElementType: SchemaTypeArray,
		},
		Items: itemType,
	}
}

func CreateRefField(ref, description string) *RefSchemaElement {
	return &RefSchemaElement{
		BaseSchemaElement: BaseSchemaElement{
			Description: description,
		},
		Ref: ref,
	}
}

func CreateObjectField(description string) *ObjectSchemaElement {
	return &ObjectSchemaElement{
		FieldSchemaElement: FieldSchemaElement{
			BaseSchemaElement: BaseSchemaElement{
				Description: description,
			},
			ElementType: SchemaTypeObject,
		},
		Properties: make(map[string]SchemaElement),
	}
}

func CreateMapField(valueType SchemaElement, description string) *ObjectSchemaElement {
	return &ObjectSchemaElement{
		FieldSchemaElement: FieldSchemaElement{
			BaseSchemaElement: BaseSchemaElement{
				Description: description,
			},
			ElementType: SchemaTypeObject,
		},
		AdditionalProperties: valueType,
	}
}

type SchemaType string

const (
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeArray   SchemaType = "array"
	SchemaTypeString  SchemaType = "string"
	SchemaTypeInteger SchemaType = "integer"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeAny     SchemaType = ""
	SchemaTypeUnknown SchemaType = "-"
)

func mergeSchemas(base SchemaObject, additional SchemaElement) error {
	if objectElement, ok := additional.(*ObjectSchemaElement); ok {
		for name, prop := range objectElement.Properties {
			base.AddProperty(name, prop)
		}
		for _, el := range objectElement.AllOf {
			base.AddEmbedded(el)
		}
		return nil
	}
	return errors.New("cannot merge non-object schema elements")
}
