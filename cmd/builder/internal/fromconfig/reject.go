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
	ContribPrefix + "extension/storage",
	ContribPrefix + "extension/observer",
	ContribPrefix + "extension/encoding",
	ContribPrefix + "extension/opampcustommessages",
	CorePrefix + "extension/extensionauth",
	CorePrefix + "extension/extensioncapabilities",
	CorePrefix + "extension/extensionmiddleware",
}

// isNotComponent reports whether a candidate module path is known to be a Go
// module that cannot host a component of the given class. Emitting such a path
// would produce a manifest that only fails at build time.
func isNotComponent(class Class, modulePath string) bool {
	if strings.Contains(modulePath, "/internal/") || strings.Contains(modulePath, "/examples/") {
		return true
	}

	// The helper, test and experimental modules sit next to the components.
	kind := string(class)
	base := path.Base(modulePath)
	if strings.HasSuffix(base, "test") || strings.HasSuffix(base, "helper") ||
		(strings.HasPrefix(base, "x") && strings.HasSuffix(base, kind)) {
		return true
	}

	// Exact matching only: a denylisted path does not reject modules nested below it.
	return slices.Contains(notComponents, modulePath)
}
