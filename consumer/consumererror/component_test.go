// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestComponent_Error(t *testing.T) {
	e := errors.New("couldn't export")
	pe := NewComponent(e, component.MustNewID("nop"))
	require.Contains(t, pe.Error(), e.Error())
}

func TestComponent_Unwrap(t *testing.T) {
	var err error = testErrorType{"some error"}
	ce := NewComponent(err, component.MustNewIDWithName("nop", "test"))
	joinedErr := errors.Join(errors.New("other error"), ce)
	target := testErrorType{}
	require.NotEqual(t, err, target)
	require.True(t, errors.As(joinedErr, &target))
	require.Equal(t, err, target)
}

func TestComponent_ID(t *testing.T) {
	id := component.MustNewIDWithName("nop", "test")
	e := NewComponent(errors.New("couldn't export"), id)
	var ce Component
	ok := errors.As(e, &ce)
	require.True(t, ok)
	require.Equal(t, id, ce.ID())
}
