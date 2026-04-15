// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionscrapercontroller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeregisterFuncNil(t *testing.T) {
	var f DeregisterFunc
	assert.NoError(t, f.Deregister(context.Background()))
}

func TestDeregisterFuncDelegates(t *testing.T) {
	called := false
	expectedErr := errors.New("deregister error")
	f := DeregisterFunc(func(context.Context) error {
		called = true
		return expectedErr
	})
	err := f.Deregister(context.Background())
	require.True(t, called)
	assert.Equal(t, expectedErr, err)
}
