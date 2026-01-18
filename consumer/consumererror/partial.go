// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

// PartialError indicates that some items in a batch were processed successfully
// while others failed. It carries the count of failed items so that observability
// infrastructure can accurately report partial success.
//
// This error type is primarily intended for use by receivers that can partially
// process incoming data. For example, a receiver that successfully processes
// 80 out of 100 metrics can return a PartialError with failedCount=20.
//
// The receiverhelper will use this information to correctly compute:
//   - accepted items = total received - failed count
//   - failed/refused items = failed count (based on error type)
type PartialError struct {
	err         error
	failedCount int
}

// NewPartialError creates an error indicating partial failure.
// failedCount is the number of items that failed to be processed.
// The wrapped error should describe what went wrong with the failed items.
//
// Example usage:
//
//	// In a receiver that processes metrics
//	if len(translationErrors) > 0 {
//	    return consumererror.NewPartialError(
//	        fmt.Errorf("failed to translate %d metrics", len(translationErrors)),
//	        len(translationErrors),
//	    )
//	}
//
// To indicate that the failure was due to downstream refusal, wrap with NewDownstream:
//
//	return consumererror.NewDownstream(
//	    consumererror.NewPartialError(err, failedCount),
//	)
func NewPartialError(err error, failedCount int) error {
	if failedCount < 0 {
		failedCount = 0
	}
	return &PartialError{
		err:         err,
		failedCount: failedCount,
	}
}

// Error implements the error interface.
func (p *PartialError) Error() string {
	return p.err.Error()
}

// Unwrap returns the wrapped error for use with errors.Is and errors.As.
func (p *PartialError) Unwrap() error {
	return p.err
}

// FailedCount returns the number of items that failed to be processed.
func (p *PartialError) FailedCount() int {
	return p.failedCount
}

// GetPartialError extracts a PartialError from an error chain if present.
// Returns nil if no PartialError is found in the error chain.
//
// This function is typically used by observability infrastructure to
// determine if an error represents a partial failure and extract the
// count of failed items.
func GetPartialError(err error) *PartialError {
	var pe *PartialError
	if errors.As(err, &pe) {
		return pe
	}
	return nil
}
