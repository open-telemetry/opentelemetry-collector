// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import "go.opentelemetry.io/collector/component"

// Component contains information about the component that returned an
// error. This is helpful when reasoning about the context an error
// occurred in when handling errors in upstream components within a
// pipeline.
type Component struct {
	err error
	id  component.ID
}

var _ error = Component{}

// NewComponent instantiates a new Component error.
// `id` should be the component's component.ID object.
func NewComponent(err error, id component.ID) error {
	return Component{err: err, id: id}
}

// Error returns a string representation of the underlying error.
func (c Component) Error() string {
	return c.id.String() + " returned an error: " + c.err.Error()
}

// Unwrap returns the wrapped error for functions Is and As in standard package errors.
func (c Component) Unwrap() error {
	return c.err
}

// Count returns the count of rejected records.
func (c Component) ID() component.ID {
	return c.id
}
