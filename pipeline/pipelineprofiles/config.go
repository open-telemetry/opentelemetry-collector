// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprofiles // import "go.opentelemetry.io/collector/pipeline/pipelineprofiles"

import (
	"go.opentelemetry.io/collector/pipeline/internal/globalsignal"
)

var (
	SignalProfiles = globalsignal.MustNewSignal("profiles")
)
