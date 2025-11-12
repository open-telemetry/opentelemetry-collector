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

// NewPartial creates a new partial error. This is used to report errors
// where only a subset of the total items failed to be written, but it
// is not possible to tell which particular items failed. An example of this
// would be a backend that can report partial successes, but only communicate
// the number of failed items without specifying which specific items failed.
//
// A partial error wraps a PermanentError; it can be treated as any other permanent
// error with no changes, meaning that consumers can transition to producing this
// error when appropriate without breaking any parts of the pipeline that check for
// permanent errors.
//
// In a scenario where the exact items that failed are known and can be retried,
// it's recommended to use the respective signal error ([consumererror.Logs],
// [consumererror.Metrics], or [consumererror.Traces]).
func NewPartial(err error, failed int) error {
	return consumererror.NewPermanent(partialError{
		inner:  err,
		failed: failed,
	})
}

// IsPartial checks if an error was wrapped with the NewPartial function,
// or if it contains one such error in its Unwrap() tree. The results are
// the count of failed items if the error is a partial error, and a boolean
// result of whether the error was a partial or not.
func IsPartial(err error) (int, bool) {
	var pe partialError
	ok := errors.As(err, &pe)
	return pe.failed, ok
}
