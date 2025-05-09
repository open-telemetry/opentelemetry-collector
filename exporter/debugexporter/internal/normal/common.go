// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// writeAttributes returns a slice of strings in the form "attrKey=attrValue"
func writeAttributes(attributes pcommon.Map) (attributeStrings []string) {
	for k, v := range attributes.All() {
		attribute := fmt.Sprintf("%s=%s", k, v.AsString())
		attributeStrings = append(attributeStrings, attribute)
	}
	return attributeStrings
}

// writeAttributesString returns a string in the form " attrKey=attrValue attr2=value2"
func writeAttributesString(attributesMap pcommon.Map) (attributesString string) {
	attributes := writeAttributes(attributesMap)
	if len(attributes) > 0 {
		attributesString = fmt.Sprintf(" %s", strings.Join(attributes, " "))
	}
	return attributesString
}

func writeResourceDetails(schemaUrl string) (resourceDetails string) {
	if len(schemaUrl) > 0 {
		resourceDetails = " [" + schemaUrl + "]"
	}
	return resourceDetails
}

func writeScopeDetails(name string, version string, schemaUrl string) (scopeDetails string) {
	if len(name) > 0 {
		scopeDetails += name
	}
	if len(version) > 0 {
		scopeDetails += "@" + version
	}
	if len(schemaUrl) > 0 {
		if len(scopeDetails) > 0 {
			scopeDetails += " "
		}
		scopeDetails += "[" + schemaUrl + "]"
	}
	if len(scopeDetails) > 0 {
		scopeDetails = " " + scopeDetails
	}
	return scopeDetails
}
