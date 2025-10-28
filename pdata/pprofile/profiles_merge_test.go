// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProfilesMergeTo(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	destLinks := dest.Dictionary().LinkTable()
	destLinks.AppendEmpty()
	existingTraceID := pcommon.TraceID([16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1})
	existingSpanID := pcommon.SpanID([8]byte{2, 2, 2, 2, 2, 2, 2, 2})
	destLink := destLinks.AppendEmpty()
	destLink.SetTraceID(existingTraceID)
	destLink.SetSpanID(existingSpanID)

	srcLinks := src.Dictionary().LinkTable()
	srcLinks.AppendEmpty()
	newTraceID := pcommon.TraceID([16]byte{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3})
	newSpanID := pcommon.SpanID([8]byte{4, 4, 4, 4, 4, 4, 4, 4})
	srcLink := srcLinks.AppendEmpty()
	srcLink.SetTraceID(newTraceID)
	srcLink.SetSpanID(newSpanID)

	sample := src.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty().
		Sample().AppendEmpty()
	sample.SetLinkIndex(1)

	require.NoError(t, src.MergeTo(dest))

	require.Equal(t, 1, dest.ResourceProfiles().Len())
	destSample := dest.ResourceProfiles().At(0).
		ScopeProfiles().At(0).
		Profiles().At(0).
		Sample().At(0)
	assert.Equal(t, int32(2), destSample.LinkIndex())

	links := dest.Dictionary().LinkTable()
	require.Equal(t, 3, links.Len())
	assert.Equal(t, existingTraceID, links.At(1).TraceID())
	assert.Equal(t, existingSpanID, links.At(1).SpanID())
	assert.Equal(t, newTraceID, links.At(2).TraceID())
	assert.Equal(t, newSpanID, links.At(2).SpanID())

	require.Equal(t, 1, src.ResourceProfiles().Len())
	srcSample := src.ResourceProfiles().At(0).
		ScopeProfiles().At(0).
		Profiles().At(0).
		Sample().At(0)
	assert.Equal(t, int32(1), srcSample.LinkIndex())
	assert.Equal(t, 2, src.Dictionary().LinkTable().Len())
}

func TestProfilesMergeToError(t *testing.T) {
	src := NewProfiles()
	dest := NewProfiles()

	stackTable := src.Dictionary().StackTable()
	stackTable.AppendEmpty()
	stack := stackTable.AppendEmpty()
	stack.LocationIndices().Append(1)

	locationTable := src.Dictionary().LocationTable()
	locationTable.AppendEmpty()
	locationTable.AppendEmpty().SetMappingIndex(1)

	sample := src.ResourceProfiles().AppendEmpty().
		ScopeProfiles().AppendEmpty().
		Profiles().AppendEmpty().
		Sample().AppendEmpty()
	sample.SetStackIndex(1)

	err := src.MergeTo(dest)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid mapping index 1")

	assert.Equal(t, 0, dest.ResourceProfiles().Len())
	assert.Equal(t, 0, dest.Dictionary().StackTable().Len())
	assert.Equal(t, 0, dest.Dictionary().LocationTable().Len())

	require.Equal(t, 1, src.ResourceProfiles().Len())
	srcSample := src.ResourceProfiles().At(0).
		ScopeProfiles().At(0).
		Profiles().At(0).
		Sample().At(0)
	assert.Equal(t, int32(1), srcSample.StackIndex())
	assert.Equal(t, 2, src.Dictionary().StackTable().Len())
	assert.Equal(t, 2, src.Dictionary().LocationTable().Len())
}
