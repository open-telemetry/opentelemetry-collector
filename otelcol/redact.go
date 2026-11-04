// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
)

// nameKey and valueKey are the serialized field names of configopaque.Pair.
const (
	nameKey  = "name"
	valueKey = "value"
)

// redactedMask is the masked value emitted by configopaque.String.MarshalText.
var redactedMask = func() string {
	b, _ := configopaque.String("").MarshalText()
	return string(b)
}()

// redactByMirroring returns a copy of raw conf where every leaf set to redactedMask
// in the redacted view is copied over, unless the corresponding pre-expansion leaf
// is exactly a ${...} provider reference that does not embed a non-empty env-provider
// literal default. A default such as ${env:NAME:-literal} is masked because the
// fallback is configuration text, not a provider lookup.
func redactByMirroring(raw, redacted *confmap.Conf) *confmap.Conf {
	if raw == nil || redacted == nil {
		return raw
	}
	redactedRaw := applyMask(raw.ToStringMap(), redacted.ToStringMap())
	return confmap.NewFromStringMap(redactedRaw.(map[string]any))
}

// applyMask recursively applies redaction to `raw` by mirroring it with the redacted structure.
// It mutates raw and returns it.
func applyMask(raw, redacted any) any {
	switch redVal := redacted.(type) {
	case map[string]any:
		if rawMap, ok := raw.(map[string]any); ok {
			for k, rawMapVal := range rawMap {
				if redMapVal, ok := redVal[k]; ok {
					rawMap[k] = applyMask(rawMapVal, redMapVal)
				}
			}
		}
	case []any:
		switch rawVal := raw.(type) {
		case []any:
			for i := 0; i < min(len(rawVal), len(redVal)); i++ {
				rawVal[i] = applyMask(rawVal[i], redVal[i])
			}
		case map[string]any:
			// configopaque.MapList accepts a map as input but marshals as a list
			// of name/value pairs. Match those pairs back to the raw map by name.
			for _, redItem := range redVal {
				redMap, ok := redItem.(map[string]any)
				if !ok || len(redMap) != 2 {
					continue
				}
				name, ok := redMap[nameKey].(string)
				if !ok {
					continue
				}
				redMapVal, ok := redMap[valueKey]
				if !ok {
					continue
				}
				rawMapVal, ok := rawVal[name]
				if ok {
					rawVal[name] = applyMask(rawMapVal, redMapVal)
				}
			}
		}
	case string:
		if redVal == redactedMask {
			if rawVal, ok := raw.(string); !ok || !preserveUnexpandedProviderReference(rawVal) {
				return redactedMask
			}
		}
	}
	return raw
}

// providerReferenceRegexp matches a leaf that is only a ${...} provider reference.
// Nested braces, such as ${env:A:-${env:B}}, do not match and stay masked.
var providerReferenceRegexp = regexp.MustCompile(`^\$\{[^}]*\}$`)

// preserveUnexpandedProviderReference reports whether an unexpanded opaque leaf
// should stay visible. A whole ${...} reference is preserved unless its scheme is
// env and the opaque value uses the env provider's ":-" default suffix with a
// non-empty literal. An empty default (${env:NAME:-}) embeds no literal and is
// preserved. Other schemes, including a file path that happens to contain ":-",
// are left unchanged. The scheme cannot contain ":", so the first colon is the split.
func preserveUnexpandedProviderReference(raw string) bool {
	if !providerReferenceRegexp.MatchString(raw) {
		return false
	}
	uri := raw[2 : len(raw)-1]
	scheme, opaque, ok := strings.Cut(uri, ":")
	if !ok || scheme != "env" {
		return true
	}
	_, def, hasDefault := strings.Cut(opaque, ":-")
	return !hasDefault || def == ""
}
