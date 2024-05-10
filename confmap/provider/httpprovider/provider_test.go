// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestSupportedScheme(t *testing.T) {
	fp := NewFactory().Create(confmaptest.NewNopProviderSettings())
	assert.Equal(t, "http", fp.Scheme())
	require.NoError(t, fp.Shutdown(context.Background()))
}
