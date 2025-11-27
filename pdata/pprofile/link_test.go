// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestLinkEqual(t *testing.T) {
	for _, tt := range []struct {
		name string
		orig Link
		dest Link
		want bool
	}{
		{
			name: "empty links",
			orig: NewLink(),
			dest: NewLink(),
			want: true,
		},
		{
			name: "non-empty identical links",
			orig: buildLink(
				pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}),
				pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			),
			dest: buildLink(
				pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}),
				pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			),
			want: true,
		},
		{
			name: "with different trace IDs",
			orig: buildLink(
				pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}),
				pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			),
			dest: buildLink(
				pcommon.TraceID([16]byte{8, 7, 6, 5, 4, 3, 2, 1, 1, 2, 3, 4, 5, 6, 7, 8}),
				pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			),
			want: false,
		},
		{
			name: "with different span IDs",
			orig: buildLink(
				pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}),
				pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}),
			),
			dest: buildLink(
				pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}),
				pcommon.SpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1}),
			),
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				assert.True(t, tt.orig.Equal(tt.dest))
			} else {
				assert.False(t, tt.orig.Equal(tt.dest))
			}
		})
	}
}

func buildLink(traceID pcommon.TraceID, spanID pcommon.SpanID) Link {
	l := NewLink()
	l.SetTraceID(traceID)
	l.SetSpanID(spanID)
	return l
}
