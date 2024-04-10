// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

type Partial struct {
	err   error
	count int
}

var _ error = &Partial{}

func NewPartial(err error, count int) error {
	return &Partial{err: err, count: count}
}

func (p *Partial) Error() string {
	return "Partial success: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p *Partial) Unwrap() error {
	return p.err
}

func (p *Partial) Count() int {
	return p.count
}
