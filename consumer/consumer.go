// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer/internal"
)

type Capabilities = internal.Capabilities

var errNilFunc = errors.New("nil consumer func")

type Option = internal.Option

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities Capabilities) Option {
	return func(o *internal.BaseImpl) {
		o.Cap = capabilities
	}
}
