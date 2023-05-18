// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExampleReceiver(t *testing.T) {
	rcv := &ExampleReceiver{}
	host := componenttest.NewNopHost()
	assert.False(t, rcv.Started())
	assert.NoError(t, rcv.Start(context.Background(), host))
	assert.True(t, rcv.Started())

	assert.False(t, rcv.Stopped())
	assert.NoError(t, rcv.Shutdown(context.Background()))
	assert.True(t, rcv.Stopped())
}
