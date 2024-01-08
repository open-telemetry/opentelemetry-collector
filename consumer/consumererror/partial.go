// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

var noDataCount = -1

type Partial struct {
	err   error
	count int
}

func NewPartial(err error, count int) error {
	return Partial{err: err, count: count}
}

func (p Partial) Error() string {
	return "Partial success: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p Partial) Unwrap() error {
	return p.err
}
