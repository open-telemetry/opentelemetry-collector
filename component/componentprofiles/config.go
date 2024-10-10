// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentprofiles // import "go.opentelemetry.io/collector/component/componentprofiles"

import (
	"go.opentelemetry.io/collector/internal/globalsignal"
)

var (
	SignalProfiles = globalsignal.MustNewSignal("profiles")
)
