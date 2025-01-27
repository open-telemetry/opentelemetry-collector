// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestFromAttributeIndices(t *testing.T) {
	profile := NewProfile()
	att := profile.AttributeTable().AppendEmpty()
	att.SetKey("hello")
	att.Value().SetStr("world")
	att2 := profile.AttributeTable().AppendEmpty()
	att2.SetKey("bonjour")
	att2.Value().SetStr("monde")

	attrs := FromAttributeIndices(profile, profile)
	assert.Equal(t, attrs, pcommon.NewMap())

	// A Location with a single attribute
	loc := NewLocation()
	loc.AttributeIndices().Append(0)

	attrs = FromAttributeIndices(profile, loc)

	m := map[string]any{"hello": "world"}
	assert.Equal(t, attrs.AsRaw(), m)

	// A Mapping with two attributes
	mapp := NewLocation()
	mapp.AttributeIndices().Append(0, 1)

	attrs = FromAttributeIndices(profile, mapp)

	m = map[string]any{"hello": "world", "bonjour": "monde"}
	assert.Equal(t, attrs.AsRaw(), m)
}

func BenchmarkFromAttributeIndices(b *testing.B) {
	profile := NewProfile()

	for i := range 10 {
		att := profile.AttributeTable().AppendEmpty()
		att.SetKey(fmt.Sprintf("key_%d", i))
		att.Value().SetStr(fmt.Sprintf("value_%d", i))
	}

	obj := NewLocation()
	obj.AttributeIndices().Append(1, 3, 7)

	b.ResetTimer()
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		_ = FromAttributeIndices(profile, obj)
	}
}
