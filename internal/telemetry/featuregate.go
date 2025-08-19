// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/internal/telemetry"

import "go.opentelemetry.io/collector/featuregate"

const (
	// NewPipelineTelemetryReceiverErrorID is the ID for the feature gate that controls whether to split failed logs/metrics/spans into refused and failed.
	NewPipelineTelemetryReceiverErrorID = "receiver.newTelemetryError"
)

// NewPipelineTelemetryReceiverError is the feature gate that controls whether to split failed metrics into refused and failed.
var NewPipelineTelemetryReceiverError = featuregate.GlobalRegistry().MustRegister(
	NewPipelineTelemetryReceiverErrorID,
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("Controls whether to split failed signal metrics into refused and failed, and to distinquish pipeline errors as downstream or internal."),
)
