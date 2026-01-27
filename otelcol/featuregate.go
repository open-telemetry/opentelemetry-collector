// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import "go.opentelemetry.io/collector/featuregate"

// receiverPartialReloadGate controls whether configuration changes that only
// affect receivers trigger a partial reload (restart receivers only) instead
// of a full service restart.
var receiverPartialReloadGate = featuregate.GlobalRegistry().MustRegister(
	"service.receiverPartialReload",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription(
		"When enabled, configuration changes that only affect receivers "+
			"will trigger a partial reload that restarts only the receivers "+
			"instead of the entire service.",
	),
)
