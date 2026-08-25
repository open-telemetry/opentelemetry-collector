// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpsprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestSupportedScheme(t *testing.T) {
	fp := NewFactory().Create(confmaptest.NewNopProviderSettings())
	assert.Equal(t, "https", fp.Scheme())
}
