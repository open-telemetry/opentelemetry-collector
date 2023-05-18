// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExampleProcessor(t *testing.T) {
	prc := &ExampleProcessor{}
	host := componenttest.NewNopHost()
	assert.False(t, prc.Started())
	assert.NoError(t, prc.Start(context.Background(), host))
	assert.True(t, prc.Started())

	assert.False(t, prc.Stopped())
	assert.NoError(t, prc.Shutdown(context.Background()))
	assert.True(t, prc.Stopped())
}
