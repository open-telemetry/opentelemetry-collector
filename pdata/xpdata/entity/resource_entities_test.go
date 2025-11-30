// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

	e := entities.PutEmpty("service")

	assert.Equal(t, 1, entities.Len())
	retrieved, ok := entities.Get("service")
	assert.True(t, ok)
	assert.Equal(t, "service", retrieved.Type())
	assert.Equal(t, "service", e.Type())
}
