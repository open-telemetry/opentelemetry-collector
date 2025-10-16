// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xpdata_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/xpdata"
)

func TestMapBuilder(t *testing.T) {
	var mb xpdata.MapBuilder
	mb.EnsureCapacity(3)
	mb.AppendEmpty("key1").SetStr("val")
	mb.AppendEmpty("key2").SetInt(42)

	m := pcommon.NewMap()
	mb.UnsafeIntoMap(m)

	assert.Equal(t, 2, m.Len())
	val, ok := m.Get("key1")
	assert.True(t, ok && val.Type() == pcommon.ValueTypeStr && val.Str() == "val")
	val, ok = m.Get("key2")
	assert.True(t, ok && val.Type() == pcommon.ValueTypeInt && val.Int() == 42)
}
