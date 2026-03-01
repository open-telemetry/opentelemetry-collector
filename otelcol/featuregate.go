// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import "go.opentelemetry.io/collector/featuregate"

// partialReloadGate controls whether configuration changes trigger partial
// reloads instead of full service restarts. When enabled, only the minimum
// necessary components are restarted based on what changed:
// - Receiver-only changes: restart receivers
// - Processor changes: restart processors and receivers
// - Exporter/connector/extension/pipeline changes: full reload
var partialReloadGate = featuregate.GlobalRegistry().MustRegister(
	"service.partialReload",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription(
		"When enabled, configuration changes trigger partial reloads "+
			"that restart only the minimum necessary components. Receiver-only "+
			"changes restart receivers; processor changes restart processors and "+
			"receivers; exporter, connector, extension, or pipeline structure "+
			"changes trigger a full reload.",
	),
	featuregate.WithRegisterFromVersion("v0.145.0"),
)
