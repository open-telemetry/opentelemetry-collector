// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownstream(t *testing.T) {
	err1 := errors.New("test error")
	err2 := errors.New("test error 2")
	assert.False(t, IsDownstream(err1))
	assert.False(t, IsDownstream(err2))

	errDownstream1 := NewDownstream(err1)
	errDownstream2 := NewDownstream(err2)
	errWrap := NewPermanent(errDownstream1)
	errJoin1 := errors.Join(errDownstream1, err2)
	errJoin2 := errors.Join(errDownstream1, errDownstream2)

	assert.Equal(t, err1.Error(), errDownstream1.Error())
	assert.True(t, errors.Is(errDownstream1, err1))
	assert.False(t, errors.Is(errDownstream1, err2))
	assert.True(t, IsDownstream(errDownstream1))

	assert.True(t, IsDownstream(errWrap))
	assert.False(t, IsDownstream(errJoin1))
	assert.True(t, IsDownstream(errJoin2))
}
