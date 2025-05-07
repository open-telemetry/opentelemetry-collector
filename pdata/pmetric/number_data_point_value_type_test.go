// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberDataPointValueTypeString(t *testing.T) {
	assert.Equal(t, "Empty", NumberDataPointValueTypeEmpty.String())
	assert.Equal(t, "Int", NumberDataPointValueTypeInt.String())
	assert.Equal(t, "Double", NumberDataPointValueTypeDouble.String())
	assert.Empty(t, (NumberDataPointValueTypeDouble + 1).String())
}
