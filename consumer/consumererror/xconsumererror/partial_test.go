// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumererror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

func TestPartial(t *testing.T) {
	internalErr := errors.New("some points failed")
	err := NewPartial(internalErr, 5)
	assert.True(t, consumererror.IsPermanent(err))
	partialErr, ok := AsPartial(err)
	assert.True(t, ok)
	assert.Equal(t, 5, partialErr.Failed())
	assert.Equal(t, internalErr, partialErr.Unwrap())
	assert.Equal(t, internalErr.Error(), partialErr.Error())
}
