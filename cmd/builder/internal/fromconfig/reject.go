// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package fromconfig // import "go.opentelemetry.io/collector/cmd/builder/internal/fromconfig"

import (
	"path"
	"slices"
	"strings"
)

// notComponents lists modules that exist under the class directories but do not
// provide a component factory.
var notComponents = []string{
	contribPrefix + "extension/storage",
	contribPrefix + "extension/observer",
	contribPrefix + "extension/encoding",
	contribPrefix + "extension/opampcustommessages",
	corePrefix + "extension/extensionauth",
	corePrefix + "extension/extensioncapabilities",
	corePrefix + "extension/extensionmiddleware",
}

// isNotComponent reports whether a candidate module path is known to be a Go
// module that cannot host a component of the given class. Emitting such a path
// would produce a manifest that only fails at build time.
func isNotComponent(class Class, modulePath string) bool {
	if strings.Contains(modulePath, "/internal/") || strings.Contains(modulePath, "/examples/") {
		return true
	}

	last := path.Base(modulePath)
	if strings.HasSuffix(last, "test") || strings.HasSuffix(last, "helper") {
		return true
	}
	if last == "x"+string(class) {
		return true
	}
	if strings.HasPrefix(last, "x") && (strings.HasSuffix(last, string(class)) || strings.HasSuffix(last, "helper")) {
		return true
	}

	// Exact matching only: a denylisted path does not reject modules nested below it.
	return slices.Contains(notComponents, modulePath)
}
