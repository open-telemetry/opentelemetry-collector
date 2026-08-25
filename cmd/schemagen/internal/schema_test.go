// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaToJSON(t *testing.T) {
	schema := CreateSchema()
	schema.Description = "An example schema for testing."
	schema.ElementType = SchemaTypeObject
	schema.AddProperty("name", CreateSimpleField(SchemaTypeString, "The name of the entity."))
	schema.AddProperty("age", CreateSimpleField(SchemaTypeInteger, "The age of the entity."))

	rawJSON, err := schema.ToJSON()
	require.NoError(t, err)

	expected := `{
		"description": "An example schema for testing.",
		"type": "object",
		"properties": {
			"name": {
				"description": "The name of the entity.",
				"type": "string"
			},
			"age": {
				"description": "The age of the entity.",
				"type": "integer"
			}
		}
	}`

	require.JSONEq(t, expected, string(rawJSON))
}

func TestSchemaWithComplexFields(t *testing.T) {
	schema := CreateSchema()
	schema.Description = "An example schema with nested struct."
	schema.ElementType = SchemaTypeObject

	addressSchema := CreateObjectField("Address object")
	addressSchema.AddProperty("street", CreateSimpleField(SchemaTypeString, "The street address."))
	addressSchema.AddProperty("city", CreateSimpleField(SchemaTypeString, "The city."))

	schema.AddProperty("address", addressSchema)
	schema.AddProperty("tags", CreateArrayField(CreateSimpleField(SchemaTypeString, "A tag."), "Array of tags."))
	schema.AddProperty("metadata", CreateRefField("#/definitions/Metadata", "Reference to Metadata definition."))

	rawJSON, err := schema.ToJSON()
	require.NoError(t, err)

	expected := `{
		"description": "An example schema with nested struct.",
		"type": "object",
		"properties": {
			"address": {
				"description": "Address object",
				"type": "object",
				"properties": {
					"street": {
						"description": "The street address.",
						"type": "string"
					},
					"city": {
						"description": "The city.",
						"type": "string"
					}
				}
			},
			"tags": {
				"description": "Array of tags.",
				"type": "array",
				"items": {
					"description": "A tag.",
					"type": "string"
				}
			},
			"metadata": {
				"description": "Reference to Metadata definition.",
				"$ref": "#/definitions/Metadata"
			}
		}
	}`

	require.JSONEq(t, expected, string(rawJSON))
}

func TestSchemaToYAML(t *testing.T) {
	schema := CreateSchema()
	schema.Description = "An example schema for testing."
	schema.ElementType = SchemaTypeObject
	schema.AddProperty("name", CreateSimpleField(SchemaTypeString, "The name of the entity."))
	schema.AddProperty("age", CreateSimpleField(SchemaTypeInteger, "The age of the entity."))

	rawYAML, err := schema.ToYAML()
	require.NoError(t, err)

	expected := `
description: An example schema for testing.
type: object
properties:
  name:
    description: The name of the entity.
    type: string
  age:
    description: The age of the entity.
    type: integer
`

	require.YAMLEq(t, expected, string(rawYAML))
}
