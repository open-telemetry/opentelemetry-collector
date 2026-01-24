// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

// partialError is an error that indicates a partial failure during export.
// Some items were successfully exported while others were dropped.
// This error is not retryable since the dropped items cannot be recovered.
type partialError struct {
	err     error
	dropped int
}

// NewPartialError creates an error that indicates a partial failure during export.
// Use this error when some items were successfully processed but others were
// intentionally dropped (e.g., due to incompatible data format).
// The dropped parameter indicates the number of items that were dropped.
// This error is treated as permanent (non-retryable) since dropped items cannot be recovered.
func NewPartialError(err error, dropped int) error {
	return partialError{err: err, dropped: dropped}
}

func (p partialError) Error() string {
	return p.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (p partialError) Unwrap() error {
	return p.err
}

// Dropped returns the number of items that were dropped during export.
func (p partialError) Dropped() int {
	return p.dropped
}

// AsPartialError checks if an error was wrapped with NewPartialError and returns
// the number of dropped items. If the error is not a partial error, returns 0 and false.
func AsPartialError(err error) (dropped int, ok bool) {
	var p partialError
	if errors.As(err, &p) {
		return p.dropped, true
	}
	return 0, false
}
