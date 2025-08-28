// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import "go.opentelemetry.io/collector/featuregate"

// NewReceiverMetricsGate is the feature gate that controls whether to distinguish downstream errors from internal errors in pipeline telemetry.
var NewReceiverMetricsGate = featuregate.GlobalRegistry().MustRegister(
	"receiverhelper.newReceiverMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.138.0"),
	featuregate.WithRegisterDescription("Controls whether receivers emit new metrics and span attributes to distinguish downstream errors from internal errors. This is a breaking change for the semantics of the otelcol_receiver_refused_metric_points,  otelcol_receiver_refused_log_records and otelcol_receiver_refused_spans."),
)
