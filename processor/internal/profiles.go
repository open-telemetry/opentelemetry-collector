// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/processor/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
)

// Profiles is a processor that can consume profiles.
type Profiles interface {
	component.Component
	consumerprofiles.Profiles
}
