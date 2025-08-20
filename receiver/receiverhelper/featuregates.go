// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import "go.opentelemetry.io/collector/featuregate"

// NewReceiverMetricsGate is the feature gate that controls whether to distinguish downstream errors from internal errors in pipeline telemetry.
var NewReceiverMetricsGate = featuregate.GlobalRegistry().MustRegister(
	"receiverhelper.newReceiverMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.138.0"),
	featuregate.WithRegisterDescription("Controls whether to distinguish downstream errors from internal errors, changing the 'outcome' for produced telemetry and splitting receiver metrics into 'refused' and 'failed' categories."),
)
