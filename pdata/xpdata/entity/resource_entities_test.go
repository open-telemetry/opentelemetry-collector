// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestResourceEntityRefs(t *testing.T) {
	res := pcommon.NewResource()
	refs := ResourceEntityRefs(res)
	assert.Equal(t, 0, refs.Len())

	ref := refs.AppendEmpty()
	ref.SetType("service")

	assert.Equal(t, 1, refs.Len())
	assert.Equal(t, "service", refs.At(0).Type())
}

func TestResourceEntities(t *testing.T) {
	res := pcommon.NewResource()
	entities := ResourceEntities(res)
	assert.Equal(t, 0, entities.Len())

	_, err := entities.AppendEmpty("service")
	require.NoError(t, err)

	assert.Equal(t, 1, entities.Len())
	assert.Equal(t, "service", entities.At(0).Type())
}
