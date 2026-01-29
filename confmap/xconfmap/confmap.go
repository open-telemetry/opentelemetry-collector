// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap // import "go.opentelemetry.io/collector/confmap/xconfmap"

import (
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
)

type ExpandedValue = internal.ExpandedValue
// ToStringMapRaw returns the raw configuration map without sanitization.
// This is an experimental API and may change or be removed in future versions.
// The returned map may change at any time without prior notice.
//
// Unlike confmap.Conf.ToStringMap(), this function does not sanitize the map
// by removing expandedValue references. This allows for configmap manipulation
// without destroying internal types.
func ToStringMapRaw(conf *confmap.Conf) map[string]any {
	return internal.ToStringMapRaw(conf)
}
