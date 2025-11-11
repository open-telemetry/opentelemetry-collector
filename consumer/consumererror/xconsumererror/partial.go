// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumererror // import "go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"

import (
	"errors"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

type partialError struct {
	inner  error
	failed int
}

var _ error = partialError{}

func (pe partialError) Error() string {
	return pe.inner.Error()
}

func (pe partialError) Unwrap() error {
	return pe.inner
}

func (pe partialError) Failed() int {
	return pe.failed
}

// NewPartial creates a new partial error. This is used by consumers
// to report errors where only a subset of the total items failed
// to be written, but it is not possible to tell which particular items
// failed.
func NewPartial(err error, failed int) error {
	return consumererror.NewPermanent(partialError{
		inner:  err,
		failed: failed,
	})
}

// AsPartial checks if an error was wrapped with the NewPartial function,
// or if it contains one such error in its Unwrap() tree.
func AsPartial(err error) (partialError, bool) {
	var pe partialError
	ok := errors.As(err, &pe)
	return pe, ok
}
