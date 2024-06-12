// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterexperimental // import "go.opentelemetry.io/collector/exporterexperimental"

import (
	"go.opentelemetry.io/collector/component" // Profiles is an exporter that can consume profiles.
	"go.opentelemetry.io/collector/consumerexperimental"
)

type Profiles interface {
	component.Component
	consumerexperimental.Profiles
}
