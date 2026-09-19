// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalProtoUnsafeNestedStringAliasesInput(t *testing.T) {
	const schemaURL = "schema-url"
	src := NewExportTraceServiceRequest()
	resourceSpans := NewResourceSpans()
	resourceSpans.SchemaUrl = schemaURL
	src.ResourceSpans = append(src.ResourceSpans, resourceSpans)

	buf := make([]byte, src.SizeProto())
	src.MarshalProto(buf)

	dest := NewExportTraceServiceRequest()
	require.NoError(t, dest.UnmarshalProtoUnsafe(buf))
	require.Len(t, dest.ResourceSpans, 1)
	assert.Equal(t, schemaURL, dest.ResourceSpans[0].SchemaUrl)

	pos := bytes.Index(buf, []byte(schemaURL))
	require.NotEqual(t, -1, pos)
	buf[pos] = 'S'

	assert.Equal(t, "Schema-url", dest.ResourceSpans[0].SchemaUrl)
}
