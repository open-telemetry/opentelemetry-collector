// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

//nolint:testifylint // Testing properties of errors, no reason to use require
func TestDownstream(t *testing.T) {
	err1 := errors.New("test error")
	assert.False(t, IsDownstream(err1))
	err2 := errors.New("test error 2")
	assert.False(t, IsDownstream(err2))

	errDownstream1 := NewDownstream(err1)
	assert.True(t, IsDownstream(errDownstream1))
	assert.Equal(t, err1.Error(), errDownstream1.Error())
	assert.ErrorIs(t, errDownstream1, err1)
	assert.NotErrorIs(t, errDownstream1, err2)

	// we can access downstream wrappers through other wrappers
	errWrapDownstream := NewRetryableError(errDownstream1)
	assert.True(t, IsDownstream(errWrapDownstream))
	errorStruct := new(Error)
	assert.ErrorAs(t, errWrapDownstream, &errorStruct)

	// we can access other wrappers through downstream wrappers
	errDownstreamWrap := NewDownstream(NewRetryableError(err1))
	assert.True(t, IsDownstream(errDownstreamWrap))
	assert.ErrorAs(t, errDownstreamWrap, &errorStruct)

	// downstream + downstream = downstream
	errJoin2 := errors.Join(errDownstream1, NewDownstream(err2))
	assert.True(t, IsDownstream(errJoin2))

	// downstream + not downstream = downstream
	errJoin1 := errors.Join(errDownstream1, err2)
	assert.True(t, IsDownstream(errJoin1))
}
