// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal // import "go.opentelemetry.io/collector/exporter/debugexporter/internal/normal"

import (
	"fmt"

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
