// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorexperimental // import "go.opentelemetry.io/collector/processorexperimental"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumerexperimental"
)

var (
	errNilNextConsumer = errors.New("nil next Consumer")
)

// Profiles is a processor that can consume profiles.
type Profiles interface {
	component.Component
	consumerexperimental.Profiles
}
