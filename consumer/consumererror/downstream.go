// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

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
func NewDownstream(err error) error {
	return downstreamError{
		inner: err,
	}
}

// IsDownstream checks if an error was wrapped with the NewDownstream function,
// or, in the case of a multi-error wrapper, that all sub-errors are tagged as
// coming from downstream.
func IsDownstream(err error) bool {
	// We want an error to be considered downstream if all sub-errors are
	// downstream, but errors.As checks for matches in _any_ sub-error, so we
	// reimplement its logic.
	if _, ok := err.(downstreamError); ok { //nolint:errorlint
		return true
	}
	if wrapper, ok := err.(interface{ Unwrap() error }); ok {
		return IsDownstream(wrapper.Unwrap())
	}
	if wrapper, ok := err.(interface{ Unwrap() []error }); ok {
		for _, suberr := range wrapper.Unwrap() {
			if !IsDownstream(suberr) {
				return false
			}
		}
		return true
	}
	return false
}
