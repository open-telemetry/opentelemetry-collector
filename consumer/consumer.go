// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"go.opentelemetry.io/collector/consumer/internal"
)

// Capabilities describes the capabilities of a Processor.
type Capabilities = internal.Capabilities

// Option to construct new consumers.
type Option = internal.Option

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities Capabilities) Option {
	return internal.WithCapabilities(capabilities)
}
