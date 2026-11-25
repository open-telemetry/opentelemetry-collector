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
// fallback is configuration text, not a provider lookup. A raw subtree with no
// counterpart in the redacted view, such as a section disabled with `enabled: false`,
// has no type information to mirror and is masked by maskUnmirrored.
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
	case nil:
		// The redacted view has nothing to mirror, e.g. a configoptional section
		// disabled with `enabled: false` or an `omitempty` field that decoded to
		// its zero value. Fail closed.
		return maskUnmirrored(raw)
	case map[string]any:
		if rawMap, ok := raw.(map[string]any); ok {
			for k, rawMapVal := range rawMap {
				rawMap[k] = applyMask(rawMapVal, redVal[k])
			}
		}
	case []any:
		switch rawVal := raw.(type) {
		case []any:
			for i := range rawVal {
				var redItem any
				if i < len(redVal) {
					redItem = redVal[i]
				}
				rawVal[i] = applyMask(rawVal[i], redItem)
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

// maskUnmirrored masks a raw subtree that has no counterpart in the redacted view.
// Since it is unknown which of its leaves are opaque, every leaf that can hold a
// literal is masked; nulls, booleans and provider references that
// preserveUnexpandedProviderReference accepts are kept. It mutates raw and returns it.
func maskUnmirrored(raw any) any {
	switch rawVal := raw.(type) {
	case nil, bool:
		return raw
	case map[string]any:
		for k, v := range rawVal {
			rawVal[k] = maskUnmirrored(v)
		}
		return rawVal
	case []any:
		for i, v := range rawVal {
			rawVal[i] = maskUnmirrored(v)
		}
		return rawVal
	case string:
		if preserveUnexpandedProviderReference(rawVal) {
			return raw
		}
	}
	return redactedMask
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
