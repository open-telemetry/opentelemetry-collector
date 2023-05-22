// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errs

var _ error = (*RequestError)(nil)

// RequestError represents an error returned during HTTP client operations
type RequestError struct {
	statusCode int
	wrapped    error
}

// NewRequestError creates a new HTTP Client Request error with the given parameters
func NewRequestError(statusCode int, wrapped error) *RequestError {
	return &RequestError{
		wrapped:    wrapped,
		statusCode: statusCode,
	}
}

func (r *RequestError) Error() string {
	return r.wrapped.Error()
}

func (r *RequestError) Unwrap() error {
	return r.wrapped
}

func (r *RequestError) StatusCode() int {
	return r.statusCode
}
