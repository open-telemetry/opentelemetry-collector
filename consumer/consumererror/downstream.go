// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "errors"

type downstreamError struct {
	inner error
}

var _ error = downstreamError{}

func (de downstreamError) Error() string {
	return de.inner.Error()
}

func (de downstreamError) Unwrap() error {
	return de.inner
}

// NewDownstream wraps an error to indicate that it is a downstream error, i.e. an
// error that does not come from the current component, but from one further downstream.
// This is used by pipeline instrumentation to determine whether an operation's outcome
// was an internal failure, or if it successfully produced data that was later refused.
// This wrapper is not intended to be used manually inside components.
func NewDownstream(err error) error {
	return downstreamError{
		inner: err,
	}
}

// IsDownstream checks if an error was wrapped with the NewDownstream function,
// or if it contains one such error in its Unwrap() tree.
func IsDownstream(err error) bool {
	var de downstreamError
	return errors.As(err, &de)
}
