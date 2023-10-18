// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenterror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

// recoverable indicates a component is a recoverable error state
type recoverable struct {
	err error
}

// NewRecoverable wraps an error to indicate that a component has encountered a recoverable error.
func NewRecoverable(err error) error {
	return recoverable{err: err}
}

func (p recoverable) Error() string {
	return "Recoverable error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p recoverable) Unwrap() error {
	return p.err
}

// IsRecoverable checks if an error was wrapped with the NewRecoverable function.
func IsRecoverable(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &recoverable{})
}
