// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProfilesMergeFromBasic(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceRP.Resource().Attributes().PutStr("resource.key", "resource.value")
	sourceRP.SetSchemaUrl("http://example.com/schema")

	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceSP.Scope().SetName("test.scope")
	sourceSP.Scope().SetVersion("1.0.0")
	sourceSP.SetSchemaUrl("http://example.com/scope-schema")

	sourceProfile := sourceSP.Profiles().AppendEmpty()
	sourceProfile.SetProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	sourceProfile.SetTime(pcommon.Timestamp(1000))
	sourceProfile.SetDuration(pcommon.Timestamp(2000))
	sourceProfile.SetPeriod(100)
	sourceProfile.SetDroppedAttributesCount(2)
	sourceProfile.SetOriginalPayloadFormat("format")
	sourceProfile.OriginalPayload().FromRaw([]byte{1, 2, 3})

	destDict := dest.Dictionary()
	destDict.StringTable().Append("existing")

	sourceDict := source.Dictionary()
	sourceStrings := sourceDict.StringTable()
	typeIdx := sourceStrings.Len()
	sourceStrings.Append("cpu")
	unitIdx := sourceStrings.Len()
	sourceStrings.Append("nanoseconds")

	periodType := sourceProfile.PeriodType()
	typeIdxInt32, err := safeInt32(typeIdx)
	require.NoError(t, err)
	unitIdxInt32, err := safeInt32(unitIdx)
	require.NoError(t, err)

	periodType.SetTypeStrindex(typeIdxInt32)
	periodType.SetUnitStrindex(unitIdxInt32)

	sampleType := sourceProfile.SampleType()
	sampleType.SetTypeStrindex(typeIdxInt32)
	sampleType.SetUnitStrindex(unitIdxInt32)

	err = dest.MergeFrom(source)
	require.NoError(t, err)

	require.Equal(t, 1, dest.ResourceProfiles().Len())
	destRP := dest.ResourceProfiles().At(0)
	assert.Equal(t, "resource.value", destRP.Resource().Attributes().AsRaw()["resource.key"])
	assert.Equal(t, "http://example.com/schema", destRP.SchemaUrl())

	require.Equal(t, 1, destRP.ScopeProfiles().Len())
	destSP := destRP.ScopeProfiles().At(0)
	assert.Equal(t, "test.scope", destSP.Scope().Name())
	assert.Equal(t, "1.0.0", destSP.Scope().Version())
	assert.Equal(t, "http://example.com/scope-schema", destSP.SchemaUrl())

	require.Equal(t, 1, destSP.Profiles().Len())
	destProfile := destSP.Profiles().At(0)
	assert.Equal(t, ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), destProfile.ProfileID())
	assert.Equal(t, pcommon.Timestamp(1000), destProfile.Time())
	assert.Equal(t, pcommon.Timestamp(2000), destProfile.Duration())
	assert.Equal(t, int64(100), destProfile.Period())
	assert.Equal(t, uint32(2), destProfile.DroppedAttributesCount())
	assert.Equal(t, "format", destProfile.OriginalPayloadFormat())
	assert.Equal(t, []byte{1, 2, 3}, destProfile.OriginalPayload().AsRaw())

	destPeriod := destProfile.PeriodType()
	assert.Equal(t, int32(1), destPeriod.TypeStrindex())
	assert.Equal(t, int32(2), destPeriod.UnitStrindex())

	destSampleType := destProfile.SampleType()
	assert.Equal(t, int32(1), destSampleType.TypeStrindex())
	assert.Equal(t, int32(2), destSampleType.UnitStrindex())

	destStrings := dest.Dictionary().StringTable()
	assert.Equal(t, 3, destStrings.Len())
	assert.Equal(t, "existing", destStrings.At(0))
	assert.Equal(t, "cpu", destStrings.At(1))
	assert.Equal(t, "nanoseconds", destStrings.At(2))
}

func TestProfilesMergeFromReadOnly(t *testing.T) {
	dest := NewProfiles()
	dest.MarkReadOnly()
	source := NewProfiles()

	err := dest.MergeFrom(source)
	assert.Equal(t, ErrDestinationReadOnly, err)

	dest2 := NewProfiles()
	source2 := NewProfiles()
	source2.MarkReadOnly()

	err = dest2.MergeFrom(source2)
	assert.Equal(t, ErrSourceReadOnly, err)
}

func TestProfilesMergeFromStringTableDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	dest.Dictionary().StringTable().Append("shared")
	dest.Dictionary().StringTable().Append("dest.only")

	source.Dictionary().StringTable().Append("shared")
	source.Dictionary().StringTable().Append("source.only")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destStrings := dest.Dictionary().StringTable()
	assert.ElementsMatch(t, []string{"shared", "dest.only", "source.only"}, destStrings.AsRaw())
}

func TestProfilesMergeFromFunctionDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.Dictionary()
	sourceDict := source.Dictionary()

	destDict.StringTable().Append("func.name")
	destDict.StringTable().Append("sys.name")
	destDict.StringTable().Append("file.name")

	sourceDict.StringTable().Append("func.name")
	sourceDict.StringTable().Append("sys.name")
	sourceDict.StringTable().Append("file.name")
	sourceDict.StringTable().Append("other.func")

	destFunc := destDict.FunctionTable().AppendEmpty()
	destFunc.SetNameStrindex(0)
	destFunc.SetSystemNameStrindex(1)
	destFunc.SetFilenameStrindex(2)
	destFunc.SetStartLine(100)

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)
	sourceFunc.SetSystemNameStrindex(1)
	sourceFunc.SetFilenameStrindex(2)
	sourceFunc.SetStartLine(100)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 4, destDict.StringTable().Len())
}

func TestProfilesMergeFromMappingDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.Dictionary()
	sourceDict := source.Dictionary()

	destDict.StringTable().Append("shared.file")
	sourceDict.StringTable().Append("shared.file")

	destMapping := destDict.MappingTable().AppendEmpty()
	destMapping.SetMemoryStart(0x1000)
	destMapping.SetMemoryLimit(0x2000)
	destMapping.SetFilenameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)
	sourceMapping.SetMemoryLimit(0x2000)
	sourceMapping.SetFilenameStrindex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.StringTable().Len())
}

func TestProfilesMergeFromLocationDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.Dictionary()
	sourceDict := source.Dictionary()

	destDict.StringTable().Append("func.name")
	sourceDict.StringTable().Append("func.name")

	destFunc := destDict.FunctionTable().AppendEmpty()
	destFunc.SetNameStrindex(0)
	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)

	destMapping := destDict.MappingTable().AppendEmpty()
	destMapping.SetMemoryStart(0x1000)
	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)

	destLoc := destDict.LocationTable().AppendEmpty()
	destLoc.SetMappingIndex(0)
	destLine := destLoc.Line().AppendEmpty()
	destLine.SetFunctionIndex(0)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.LocationTable().Len())
	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 1, destDict.MappingTable().Len())
}

func TestProfilesMergeFromStackDedupAndSampleMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.Dictionary()
	sourceDict := source.Dictionary()

	destFunc := destDict.FunctionTable().AppendEmpty()
	destFunc.SetNameStrindex(-1)

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(-1)

	destLoc := destDict.LocationTable().AppendEmpty()
	destLoc.SetMappingIndex(-1)
	destLoc.Line().AppendEmpty().SetFunctionIndex(0)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(-1)
	sourceLoc.Line().AppendEmpty().SetFunctionIndex(0)

	destStack := destDict.StackTable().AppendEmpty()
	destStack.LocationIndices().Append(0)

	sourceStack := sourceDict.StackTable().AppendEmpty()
	sourceStack.LocationIndices().Append(0)

	sourceProfile := source.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetStackIndex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.StackTable().Len())
	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	require.Equal(t, 1, destProfile.Sample().Len())
	assert.Equal(t, int32(0), destProfile.Sample().At(0).StackIndex())
}

func TestProfilesMergeFromLinkDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destLink := dest.Dictionary().LinkTable().AppendEmpty()
	destLink.SetTraceID([16]byte{1})

	sourceLink := source.Dictionary().LinkTable().AppendEmpty()
	sourceLink.SetTraceID([16]byte{1})

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, dest.Dictionary().LinkTable().Len())
}

func TestProfilesMergeFromAttributeDedup(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.Dictionary()
	sourceDict := source.Dictionary()

	destDict.StringTable().Append("attr.key")
	destAttr := destDict.AttributeTable().AppendEmpty()
	destAttr.SetKeyStrindex(0)
	destAttr.Value().SetStr("value")
	destAttr.SetUnitStrindex(-1)

	sourceDict.StringTable().Append("attr.key")
	sourceDict.StringTable().Append("unit")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKeyStrindex(0)
	sourceAttr.Value().SetStr("value")
	sourceAttr.SetUnitStrindex(1)

	sourceProfile := source.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	sourceProfile.AttributeIndices().Append(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.AttributeTable().Len())
	attr := destDict.AttributeTable().At(0)
	assert.Equal(t, "value", attr.Value().Str())
	assert.Equal(t, int32(1), attr.UnitStrindex())

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	require.Equal(t, 1, destProfile.AttributeIndices().Len())
	assert.Equal(t, int32(0), destProfile.AttributeIndices().At(0))

	strings := destDict.StringTable()
	assert.Equal(t, []string{"attr.key", "unit"}, strings.AsRaw())
}

func TestProfilesMergeFromCommentStringRemap(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	dest.Dictionary().StringTable().Append("existing")

	source.Dictionary().StringTable().Append("comment")

	sourceProfile := source.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	sourceProfile.CommentStrindices().Append(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	require.Equal(t, 1, destProfile.CommentStrindices().Len())
	assert.Equal(t, int32(1), destProfile.CommentStrindices().At(0))
	assert.Equal(t, []string{"existing", "comment"}, dest.Dictionary().StringTable().AsRaw())
}
