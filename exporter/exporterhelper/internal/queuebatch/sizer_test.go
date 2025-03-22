// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestItemsSizer(t *testing.T) {
	sz := NewItemsSizer()
	assert.EqualValues(t, 3, sz.Sizeof(&requesttest.FakeRequest{Items: 3}))
}
