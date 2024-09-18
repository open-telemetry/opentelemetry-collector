// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumer // import "go.opentelemetry.io/collector/consumer"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer/internal"
)

// Capabilities describes the capabilities of a Processor.
type Capabilities = internal.Capabilities

// ObsReport describes the observability report of a consumer.
type ObsReport = internal.ObsReport

var errNilFunc = errors.New("nil consumer func")

// Option to construct new consumers.
type Option = internal.Option

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities Capabilities) Option {
	return internal.OptionFunc(func(o *internal.BaseImpl) {
		o.Cap = capabilities
	})
}

func WithObsReport(report ObsReport) Option {
	return func(o *internal.BaseImpl) {
		o.ObsReport = report
	}
}
