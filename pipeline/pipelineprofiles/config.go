// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/pipeline/xpipeline instead.
package pipelineprofiles // import "go.opentelemetry.io/collector/pipeline/pipelineprofiles"

import (
	"go.opentelemetry.io/collector/pipeline/internal/globalsignal"
)

// Deprecated: [0.116.0] Use xpipeline.SignalProfiles instead.
var SignalProfiles = globalsignal.MustNewSignal("profiles")
