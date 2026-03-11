// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"

import "strings"

func redactHeaderPath(config any, path []string) {
	if len(path) == 0 {
		return
	}
	next, rest := path[0], path[1:]
	if next == "*" {
		if configArray, ok := config.([]any); ok {
			for _, elem := range configArray {
				redactHeaderPath(elem, rest)
			}
		}
	} else if configMap, ok := config.(map[string]any); ok {
		for _, nextKey := range strings.Split(next, "|") {
			if len(path) == 1 {
				configMap[nextKey] = "[REDACTED]"
			} else if elem, ok := configMap[nextKey]; ok {
				redactHeaderPath(elem, rest)
			}
		}
	}
}

func redactHeaders(config any, path string) {
	redactHeaderPath(config, strings.Split(path, "."))
}
