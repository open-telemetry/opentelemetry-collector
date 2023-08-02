// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package component

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusEventOK(t *testing.T) {
	event, err := NewStatusEvent(StatusOK)
	require.NoError(t, err)
	require.Equal(t, StatusOK, event.Status())
	require.Nil(t, event.Err())
}

func TestStatusEventOKWithError(t *testing.T) {
	event, err := NewStatusEvent(StatusOK, WithError(errors.New("an error")))
	require.Error(t, err)
	require.Nil(t, event)
}

func TestStatusEventError(t *testing.T) {
	eventErr := errors.New("an error")
	event, err := NewStatusEvent(StatusError, WithError(eventErr))
	require.NoError(t, err)
	require.Equal(t, StatusError, event.Status())
	require.Equal(t, eventErr, event.Err())
}
