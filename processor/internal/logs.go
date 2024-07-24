// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/processor/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
)

// Logs is a processor that can consume logs.
type Logs interface {
	component.Component
	consumer.Logs
}
