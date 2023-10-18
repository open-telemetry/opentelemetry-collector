// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenterror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

// fatal indicates a component is an error state that it cannot recover from and the collector
// should terminate as a result
type fatal struct {
	err error
}

// NewFatal wraps an error to indicate that a component has encountered a fatal error.
// i.e. it is in an error state that it cannot recover from and the collector should terminate.
func NewFatal(err error) error {
	return fatal{err: err}
}

func (p fatal) Error() string {
	return "Fatal error: " + p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p fatal) Unwrap() error {
	return p.err
}

// IsFatal checks if an error was wrapped with the NewFatal function.
func IsFatal(err error) bool {
	if err == nil {
		return false
	}
	return errors.As(err, &fatal{})
}
