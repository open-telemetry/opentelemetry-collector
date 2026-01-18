// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivererror // import "go.opentelemetry.io/collector/receiver/receivererror"

import "errors"

// PartialReceiveError is an error to represent that a subset of received data
// failed to be processed by the receiver (e.g., due to translation errors,
// validation failures, or other internal receiver issues).
//
// When returned from a receiver's data processing, the observability helper
// uses the Failed count to correctly report partial success metrics:
// - accepted items = total items - Failed
// - failed items = Failed
type PartialReceiveError struct {
	error
	// Failed is the number of items that failed to be processed.
	// This count should not exceed the total number of items received.
	Failed int
}

// NewPartialReceiveError creates a PartialReceiveError for failed data.
// Use this error type only when a subset of data failed to be processed
// by the receiver. The failed parameter indicates the number of items
// that could not be processed.
func NewPartialReceiveError(err error, failed int) error {
	return PartialReceiveError{
		error:  err,
		Failed: failed,
	}
}

// Unwrap returns the wrapped error for use by errors.Is and errors.As.
func (e PartialReceiveError) Unwrap() error {
	return e.error
}

// IsPartialReceiveError checks if an error was created with NewPartialReceiveError
// or contains one such error in its Unwrap() tree.
func IsPartialReceiveError(err error) bool {
	var partialErr PartialReceiveError
	return errors.As(err, &partialErr)
}

// FailedItemsFromError extracts the failed item count from an error.
// If the error is not a PartialReceiveError, returns 0 and false.
func FailedItemsFromError(err error) (int, bool) {
	var partialErr PartialReceiveError
	if errors.As(err, &partialErr) {
		return partialErr.Failed, true
	}
	return 0, false
}
