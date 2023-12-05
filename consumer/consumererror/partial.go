// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

// Partial describes situations where only some telemetry data was
// accepted.
// The semantics for this error are based on OTLP partial success
// messages.
// See: https://github.com/open-telemetry/opentelemetry-proto/blob/9d139c87b52669a3e2825b835dd828b57a455a55/docs/specification.md#partial-success
type Partial struct {
	err   error
	count int
}

var _ error = Partial{}

// NewPartial instantiates a new Partial error.
// `count` should be a positive number representing
// the number of failed records, i.e. spans, data points,
// or log records.
func NewPartial(err error, count int) error {
	return Partial{err: err, count: count}
}

// Error returns a string representation of the underlying error.
func (p Partial) Error() string {
	return "Partial success: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p Partial) Unwrap() error {
	return p.err
}

// Count returns the count of rejected records.
func (p Partial) Count() int {
	return p.count
}
