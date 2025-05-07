// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExampleReceiver(t *testing.T) {
	rcv := &ExampleReceiver{}
	host := componenttest.NewNopHost()
	assert.False(t, rcv.Started())
	require.NoError(t, rcv.Start(context.Background(), host))
	assert.True(t, rcv.Started())

	assert.False(t, rcv.Stopped())
	require.NoError(t, rcv.Shutdown(context.Background()))
	assert.True(t, rcv.Stopped())
}
