// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExampleConnector(t *testing.T) {
	conn := &ExampleConnector{}
	host := componenttest.NewNopHost()
	assert.False(t, conn.Started())
	assert.NoError(t, conn.Start(context.Background(), host))
	assert.True(t, conn.Started())

	assert.False(t, conn.Stopped())
	assert.NoError(t, conn.Shutdown(context.Background()))
	assert.True(t, conn.Stopped())
}
