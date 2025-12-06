// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package schemagen // import "go.opentelemetry.io/collector/cmd/builder/internal/schemagen"

import (
	"go/types"
	"strings"
)

// HandleSpecialType checks if a type is a well-known special type and returns its schema.
// Returns the schema and true if it's a special type, or nil and false otherwise.
// NOTE: configoptional.Optional is NOT handled here - use GetOptionalInnerType instead.
func HandleSpecialType(t types.Type) (map[string]any, bool) {
	named, ok := t.(*types.Named)
	if !ok {
		return nil, false
	}

	pkgPath := ""
	if named.Obj().Pkg() != nil {
		pkgPath = named.Obj().Pkg().Path()
	}
	typeName := named.Obj().Name()

	// Handle time.Duration
	if typeName == "Duration" && strings.HasSuffix(pkgPath, "time") {
		return map[string]any{
			"type":        "string",
			"pattern":     "^[0-9]+(ns|us|µs|ms|s|m|h)$",
			"description": "Duration string (e.g., '1s', '5m', '1h')",
		}, true
	}

	// Handle time.Time
	if typeName == "Time" && strings.HasSuffix(pkgPath, "time") {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}, true
	}

	// Handle configopaque.String (sensitive data like passwords, API keys)
	if typeName == "String" && strings.Contains(pkgPath, "configopaque") {
		return map[string]any{
			"type": "string",
		}, true
	}

	return nil, false
}

// GetOptionalInnerType checks if a type is configoptional.Optional[T] and returns the inner type T.
// Returns the inner type and true if it's an Optional, or nil and false otherwise.
// This allows the caller to properly generate the schema for the inner type using full type analysis.
func GetOptionalInnerType(t types.Type) (types.Type, bool) {
	named, ok := t.(*types.Named)
	if !ok {
		return nil, false
	}

	pkgPath := ""
	if named.Obj().Pkg() != nil {
		pkgPath = named.Obj().Pkg().Path()
	}
	typeName := named.Obj().Name()

	if strings.HasPrefix(typeName, "Optional") && strings.Contains(pkgPath, "configoptional") {
		typeArgs := named.TypeArgs()
		if typeArgs != nil && typeArgs.Len() > 0 {
			return typeArgs.At(0), true
		}
	}
	return nil, false
}

// IsDeprecatedFromDescription checks if a field description indicates deprecation.
func IsDeprecatedFromDescription(description string) bool {
	if description == "" {
		return false
	}

	lowerDesc := strings.ToLower(description)
	keywords := []string{"deprecated", "obsolete", "legacy", "do not use", "will be removed", "replaced by"}
	for _, keyword := range keywords {
		if strings.Contains(lowerDesc, keyword) {
			return true
		}
	}
	return false
}

// IsDeprecatedFromTag checks if a struct tag indicates deprecation.
func IsDeprecatedFromTag(tag string) bool {
	if tag == "" {
		return false
	}

	// Check for explicit deprecated tag
	if strings.Contains(tag, `deprecated:`) {
		return true
	}

	// Check for deprecated in mapstructure or json tags
	if strings.Contains(tag, `mapstructure:"`) {
		for part := range strings.SplitSeq(extractTagValue(tag, "mapstructure"), ",") {
			if strings.TrimSpace(part) == "deprecated" {
				return true
			}
		}
	}

	if strings.Contains(tag, `json:"`) {
		for part := range strings.SplitSeq(extractTagValue(tag, "json"), ",") {
			if strings.TrimSpace(part) == "deprecated" {
				return true
			}
		}
	}

	return false
}

// extractTagValue extracts the value for a specific tag key.
func extractTagValue(tag, key string) string {
	prefix := key + `:"`
	idx := strings.Index(tag, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	end := strings.Index(tag[start:], `"`)
	if end == -1 {
		return ""
	}

	return tag[start : start+end]
}
