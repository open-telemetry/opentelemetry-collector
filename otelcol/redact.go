// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"regexp"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
)

// redactedMask is the masked value emitted by configopaque.String.MarshalText.
var redactedMask = func() string {
	b, _ := configopaque.String("").MarshalText()
	return string(b)
}()

// redactByMirroring returns a copy of raw conf where every leaf set to redactedMask
// in the redacted view is copied over, unless the corresponding pre-expansion leaf
// is exactly a ${...} provider reference, in which case it is left untouched.
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
				if !ok {
					continue
				}
				name, ok := redMap["name"].(string)
				if !ok {
					continue
				}
				redMapVal, ok := redMap["value"]
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
			if rawVal, ok := raw.(string); !ok || !providerReferenceRegexp.MatchString(rawVal) {
				return redactedMask
			}
		}
	}
	return raw
}

var providerReferenceRegexp = regexp.MustCompile(`^\$\{[^}]*\}$`)
