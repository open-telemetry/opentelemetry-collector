// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenterror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

// permanent indicates a component is an error state that it cannot recover from
type permanent struct {
	err error
}

// NewPermanent wraps an error to indicate that a component has encountered a permanent error.
// i.e. it is in an error state that it cannot recover from.
func NewPermanent(err error) error {
	return permanent{err: err}
}

func (p permanent) Error() string {
	return "Permanent error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p permanent) Unwrap() error {
	return p.err
}

// IsPermanent checks if an error was wrapped with the NewPermanent function.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &permanent{})
}
