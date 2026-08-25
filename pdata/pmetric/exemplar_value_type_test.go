// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExemplarValueTypeString(t *testing.T) {
	assert.Equal(t, "Empty", ExemplarValueTypeEmpty.String())
	assert.Equal(t, "Int", ExemplarValueTypeInt.String())
	assert.Equal(t, "Double", ExemplarValueTypeDouble.String())
	assert.Empty(t, (ExemplarValueTypeDouble + 1).String())
}
