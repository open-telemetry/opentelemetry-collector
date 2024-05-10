// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func TestNewNopHost(t *testing.T) {
	nh := NewNopHost()
	require.NotNil(t, nh)
	require.IsType(t, &nopHost{}, nh)

	assert.Nil(t, nh.GetExtensions())
	assert.Nil(t, nh.GetFactory(component.KindReceiver, component.MustNewType("test")))
}
