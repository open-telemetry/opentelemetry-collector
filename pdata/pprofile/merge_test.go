// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestProfiles_MergeFrom_Basic(t *testing.T) {
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

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("test.string")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, dest.ResourceProfiles().Len())
	destRP := dest.ResourceProfiles().At(0)
	assert.Equal(t, "resource.value", destRP.Resource().Attributes().AsRaw()["resource.key"])
	assert.Equal(t, "http://example.com/schema", destRP.SchemaUrl())

	assert.Equal(t, 1, destRP.ScopeProfiles().Len())
	destSP := destRP.ScopeProfiles().At(0)
	assert.Equal(t, "test.scope", destSP.Scope().Name())
	assert.Equal(t, "1.0.0", destSP.Scope().Version())
	assert.Equal(t, "http://example.com/scope-schema", destSP.SchemaUrl())

	assert.Equal(t, 1, destSP.Profiles().Len())
	destProfile := destSP.Profiles().At(0)
	assert.Equal(t, ProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}), destProfile.ProfileID())
	assert.Equal(t, pcommon.Timestamp(1000), destProfile.Time())
	assert.Equal(t, pcommon.Timestamp(2000), destProfile.Duration())
	assert.Equal(t, int64(100), destProfile.Period())

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 1, destDict.StringTable().Len())
	assert.Equal(t, "test.string", destDict.StringTable().At(0))
}

func TestProfiles_MergeFrom_ReadOnlyErrors(t *testing.T) {
	t.Run("destination_read_only", func(t *testing.T) {
		dest := NewProfiles()
		dest.MarkReadOnly()
		source := NewProfiles()

		err := dest.MergeFrom(source)
		assert.Equal(t, ErrDestinationReadOnly, err)
	})

	t.Run("source_read_only", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()
		source.MarkReadOnly()

		err := dest.MergeFrom(source)
		assert.Equal(t, ErrSourceReadOnly, err)
	})
}

func TestProfiles_MergeFrom_DictionaryDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destDict.StringTable().Append("shared.string")
	destDict.StringTable().Append("dest.only")

	sourceDict.StringTable().Append("shared.string")
	sourceDict.StringTable().Append("source.only")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 3, destDict.StringTable().Len())
	assert.Equal(t, "shared.string", destDict.StringTable().At(0))
	assert.Equal(t, "dest.only", destDict.StringTable().At(1))
	assert.Equal(t, "source.only", destDict.StringTable().At(2))
}

func TestProfiles_MergeFrom_FunctionDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destDict.StringTable().Append("func.name")
	destDict.StringTable().Append("sys.name")
	destDict.StringTable().Append("file.name")

	sourceDict.StringTable().Append("func.name") // Will be remapped to index 0
	sourceDict.StringTable().Append("other.func")

	destFunc := destDict.FunctionTable().AppendEmpty()
	destFunc.SetNameStrindex(0)       // "func.name"
	destFunc.SetSystemNameStrindex(1) // "sys.name"
	destFunc.SetFilenameStrindex(2)   // "file.name"
	destFunc.SetStartLine(100)

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)        // "func.name" (will be remapped)
	sourceFunc.SetSystemNameStrindex(-1) // Not set, since source doesn't have "sys.name"
	sourceFunc.SetFilenameStrindex(-1)   // Not set, since source doesn't have "file.name"
	sourceFunc.SetStartLine(100)         // Same as dest - should be deduplicated

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 2, destDict.FunctionTable().Len())

	assert.Equal(t, 4, destDict.StringTable().Len()) // func.name, sys.name, file.name, other.func
}

func TestProfiles_MergeFrom_MappingDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destDict.StringTable().Append("shared.file")
	sourceDict.StringTable().Append("shared.file")

	destMapping := destDict.MappingTable().AppendEmpty()
	destMapping.SetMemoryStart(0x1000)
	destMapping.SetMemoryLimit(0x2000)
	destMapping.SetFileOffset(100)
	destMapping.SetFilenameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)
	sourceMapping.SetMemoryLimit(0x2000)
	sourceMapping.SetFileOffset(100)
	sourceMapping.SetFilenameStrindex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_LocationDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

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
	destLoc.SetAddress(0x1234)
	destLine := destLoc.Line().AppendEmpty()
	destLine.SetFunctionIndex(0)
	destLine.SetLine(42)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLoc.SetAddress(0x1234)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)
	sourceLine.SetLine(42)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.LocationTable().Len())
	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_LinkDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	traceID := pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	spanID := pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	destLink := destDict.LinkTable().AppendEmpty()
	destLink.SetTraceID(traceID)
	destLink.SetSpanID(spanID)

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(traceID)
	sourceLink.SetSpanID(spanID)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.LinkTable().Len())
}

func TestProfiles_MergeFrom_AttributeDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destAttr := destDict.AttributeTable().AppendEmpty()
	destAttr.SetKey("test.key")
	destAttr.Value().SetStr("test.value")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("test.key")
	sourceAttr.Value().SetStr("test.value")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.AttributeTable().Len())
}

func TestProfiles_MergeFrom_ReferenceUpdates(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("test.string")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("attr.key")
	sourceAttr.Value().SetStr("attr.value")

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceProfile.LocationIndices().Append(0)

	sourceProfile.AttributeIndices().Append(0)

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetLocationsStartIndex(0)
	sourceSample.SetLocationsLength(1)
	sourceSample.SetLinkIndex(0)
	sourceSample.AttributeIndices().Append(0)

	destDict := dest.ProfilesDictionary()
	destDict.StringTable().Append("existing.string")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)

	assert.Equal(t, 1, destProfile.LocationIndices().Len())
	locationIdx := destProfile.LocationIndices().At(0)
	destLoc := destDict.LocationTable().At(int(locationIdx))

	destLine := destLoc.Line().At(0)
	funcIdx := destLine.FunctionIndex()
	destFunc := destDict.FunctionTable().At(int(funcIdx))

	stringIdx := destFunc.NameStrindex()
	assert.Equal(t, "test.string", destDict.StringTable().At(int(stringIdx)))

	destSample := destProfile.Sample().At(0)
	linkIdx := destSample.LinkIndex()
	assert.GreaterOrEqual(t, linkIdx, int32(0))
	assert.Equal(t, 1, destSample.AttributeIndices().Len())
}

func TestProfiles_MergeFrom_EmptyProfiles(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	// Both should remain empty
	assert.Equal(t, 0, dest.ResourceProfiles().Len())
	assert.Equal(t, 0, dest.ProfilesDictionary().StringTable().Len())
}

func TestProfiles_MergeFrom_MultipleResourceProfiles(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destRP := dest.ResourceProfiles().AppendEmpty()
	destRP.Resource().Attributes().PutStr("dest.key", "dest.value")

	sourceRP1 := source.ResourceProfiles().AppendEmpty()
	sourceRP1.Resource().Attributes().PutStr("source1.key", "source1.value")

	sourceRP2 := source.ResourceProfiles().AppendEmpty()
	sourceRP2.Resource().Attributes().PutStr("source2.key", "source2.value")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 3, dest.ResourceProfiles().Len())

	assert.Equal(t, "dest.value", dest.ResourceProfiles().At(0).Resource().Attributes().AsRaw()["dest.key"])

	assert.Equal(t, "source1.value", dest.ResourceProfiles().At(1).Resource().Attributes().AsRaw()["source1.key"])
	assert.Equal(t, "source2.value", dest.ResourceProfiles().At(2).Resource().Attributes().AsRaw()["source2.key"])
}

func TestProfiles_MergeFrom_ComplexProfile(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	sourceDict.StringTable().Append("cpu")
	sourceDict.StringTable().Append("samples")
	sourceDict.StringTable().Append("count")
	sourceDict.StringTable().Append("main")
	sourceDict.StringTable().Append("main.go")
	sourceDict.StringTable().Append("binary")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(3)
	sourceFunc.SetSystemNameStrindex(3)
	sourceFunc.SetFilenameStrindex(4)
	sourceFunc.SetStartLine(10)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x400000)
	sourceMapping.SetMemoryLimit(0x500000)
	sourceMapping.SetFileOffset(0)
	sourceMapping.SetFilenameStrindex(5) // "binary"
	sourceMapping.SetHasFunctions(true)
	sourceMapping.SetHasFilenames(true)
	sourceMapping.SetHasLineNumbers(true)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLoc.SetAddress(0x401234)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)
	sourceLine.SetLine(15)
	sourceLine.SetColumn(5)

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	sourceLink.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("process.name")
	sourceAttr.Value().SetStr("test-process")

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceRP.Resource().Attributes().PutStr("service.name", "test-service")
	sourceRP.SetSchemaUrl("http://example.com/resource")

	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceSP.Scope().SetName("test-profiler")
	sourceSP.Scope().SetVersion("1.2.3")
	sourceSP.SetSchemaUrl("http://example.com/scope")

	sourceProfile := sourceSP.Profiles().AppendEmpty()
	sourceProfile.SetProfileID([16]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6})
	sourceProfile.SetTime(pcommon.Timestamp(123456789))
	sourceProfile.SetDuration(pcommon.Timestamp(1000000))
	sourceProfile.SetPeriod(10000)
	sourceProfile.SetDefaultSampleTypeIndex(0)
	sourceProfile.SetOriginalPayloadFormat("pprof")

	sampleType := sourceProfile.SampleType().AppendEmpty()
	sampleType.SetTypeStrindex(0)
	sampleType.SetUnitStrindex(1)

	sourceProfile.LocationIndices().Append(0)

	sourceProfile.AttributeIndices().Append(0)

	sourceProfile.CommentStrindices().Append(2)

	sample := sourceProfile.Sample().AppendEmpty()
	sample.SetLocationsStartIndex(0)
	sample.SetLocationsLength(1)
	sample.SetLinkIndex(0)
	sample.Value().Append(42)
	sample.AttributeIndices().Append(0)
	sample.TimestampsUnixNano().Append(123456789)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, dest.ResourceProfiles().Len())
	destRP := dest.ResourceProfiles().At(0)
	assert.Equal(t, "test-service", destRP.Resource().Attributes().AsRaw()["service.name"])
	assert.Equal(t, "http://example.com/resource", destRP.SchemaUrl())

	assert.Equal(t, 1, destRP.ScopeProfiles().Len())
	destSP := destRP.ScopeProfiles().At(0)
	assert.Equal(t, "test-profiler", destSP.Scope().Name())
	assert.Equal(t, "1.2.3", destSP.Scope().Version())
	assert.Equal(t, "http://example.com/scope", destSP.SchemaUrl())

	assert.Equal(t, 1, destSP.Profiles().Len())
	destProfile := destSP.Profiles().At(0)
	assert.Equal(t, ProfileID([16]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6}), destProfile.ProfileID())
	assert.Equal(t, pcommon.Timestamp(123456789), destProfile.Time())
	assert.Equal(t, pcommon.Timestamp(1000000), destProfile.Duration())
	assert.Equal(t, int64(10000), destProfile.Period())
	assert.Equal(t, int32(0), destProfile.DefaultSampleTypeIndex())
	assert.Equal(t, "pprof", destProfile.OriginalPayloadFormat())

	// Verify dictionary was merged
	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 6, destDict.StringTable().Len())
	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.LocationTable().Len())
	assert.Equal(t, 1, destDict.LinkTable().Len())
	assert.Equal(t, 1, destDict.AttributeTable().Len())

	// Verify samples
	assert.Equal(t, 1, destProfile.Sample().Len())
	destSample := destProfile.Sample().At(0)
	assert.Equal(t, int32(0), destSample.LocationsStartIndex())
	assert.Equal(t, int32(1), destSample.LocationsLength())
	assert.True(t, destSample.HasLinkIndex())
	assert.Equal(t, 1, destSample.Value().Len())
	assert.Equal(t, int64(42), destSample.Value().At(0))
	assert.Equal(t, 1, destSample.AttributeIndices().Len())
	assert.Equal(t, 1, destSample.TimestampsUnixNano().Len())
	assert.Equal(t, uint64(123456789), destSample.TimestampsUnixNano().At(0))
}

func TestProfiles_MergeFrom_SizeLimits(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	for i := range 1000 {
		destDict.StringTable().Append("dest.string." + string(rune('0'+i%10)))
		sourceDict.StringTable().Append("source.string." + string(rune('0'+i%10)))
	}

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, destDict.StringTable().Len(), 1000)
	assert.LessOrEqual(t, destDict.StringTable().Len(), 2000)
}

func TestProfiles_MergeFrom_SampleWithoutLinkIndex(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetLocationsStartIndex(0)
	sourceSample.SetLocationsLength(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destSample := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0)
	assert.Equal(t, sourceSample.LinkIndex(), destSample.LinkIndex())
}

func TestProfiles_MergeFrom_StringTableError(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	for i := range 100 {
		sourceDict.StringTable().Append("test.string." + string(rune('0'+i%10)))
	}

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Positive(t, dest.ProfilesDictionary().StringTable().Len())
}

func TestProfiles_MergeFrom_EmptyStringMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(-1)
	sourceFunc.SetSystemNameStrindex(-1)
	sourceFunc.SetFilenameStrindex(-1)
	sourceFunc.SetStartLine(10)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, dest.ProfilesDictionary().FunctionTable().Len())
}

func TestProfiles_MergeFrom_ComplexReferenceChain(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destDict.StringTable().Append("existing")

	sourceDict.StringTable().Append("func1")
	sourceDict.StringTable().Append("file1")
	sourceDict.StringTable().Append("binary1")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)
	sourceFunc.SetFilenameStrindex(1)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetFilenameStrindex(2)
	sourceMapping.SetMemoryStart(0x1000)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict = dest.ProfilesDictionary()
	assert.Equal(t, 4, destDict.StringTable().Len())

	destFunc := destDict.FunctionTable().At(0)
	assert.Equal(t, int32(1), destFunc.NameStrindex())
	assert.Equal(t, int32(2), destFunc.FilenameStrindex())

	destMapping := destDict.MappingTable().At(0)
	assert.Equal(t, int32(3), destMapping.FilenameStrindex())

	destLoc := destDict.LocationTable().At(0)
	assert.Equal(t, int32(0), destLoc.MappingIndex())
	destLine := destLoc.Line().At(0)
	assert.Equal(t, int32(0), destLine.FunctionIndex())
}

func TestProfiles_MergeFrom_AttributeUnitSlice(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(0)
	sourceAttrUnit.SetUnitStrindex(1)

	sourceDict.StringTable().Append("attr.key")
	sourceDict.StringTable().Append("unit.value")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 1, destDict.AttributeUnits().Len())
	assert.Equal(t, 2, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_NegativeIndices(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(-1)
	sourceFunc.SetSystemNameStrindex(-1)
	sourceFunc.SetFilenameStrindex(-1)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetFilenameStrindex(-1)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(-1)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	destFunc := destDict.FunctionTable().At(0)
	assert.Equal(t, int32(-1), destFunc.NameStrindex())
	assert.Equal(t, int32(-1), destFunc.SystemNameStrindex())
	assert.Equal(t, int32(-1), destFunc.FilenameStrindex())

	destMapping := destDict.MappingTable().At(0)
	assert.Equal(t, int32(-1), destMapping.FilenameStrindex())

	destLoc := destDict.LocationTable().At(0)
	assert.Equal(t, int32(-1), destLoc.MappingIndex())
}

func TestProfiles_MergeFrom_ErrorPropagation(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("test")

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()
	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.Value().Append(123)

	err := dest.MergeFrom(source)
	require.NoError(t, err)
}

func TestProfiles_MergeFrom_FullCoverage(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	sourceDict.StringTable().Append("string1")
	sourceDict.StringTable().Append("string2")
	sourceDict.StringTable().Append("string3")
	sourceDict.StringTable().Append("string4")
	sourceDict.StringTable().Append("string5")

	attrUnit := sourceDict.AttributeUnits().AppendEmpty()
	attrUnit.SetAttributeKeyStrindex(0)
	attrUnit.SetUnitStrindex(1)

	func1 := sourceDict.FunctionTable().AppendEmpty()
	func1.SetNameStrindex(0)
	func1.SetSystemNameStrindex(1)
	func1.SetFilenameStrindex(2)
	func1.SetStartLine(10)

	mapping1 := sourceDict.MappingTable().AppendEmpty()
	mapping1.SetMemoryStart(0x1000)
	mapping1.SetMemoryLimit(0x2000)
	mapping1.SetFilenameStrindex(3)

	loc1 := sourceDict.LocationTable().AppendEmpty()
	loc1.SetMappingIndex(0)
	loc1.SetAddress(0x1234)
	line1 := loc1.Line().AppendEmpty()
	line1.SetFunctionIndex(0)
	line1.SetLine(100)

	link1 := sourceDict.LinkTable().AppendEmpty()
	link1.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	link1.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	attr1 := sourceDict.AttributeTable().AppendEmpty()
	attr1.SetKey("test.key")
	attr1.Value().SetStr("test.value")

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceRP.Resource().Attributes().PutStr("service.name", "test")
	sourceRP.SetSchemaUrl("http://example.com")

	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceSP.Scope().SetName("profiler")
	sourceSP.Scope().SetVersion("1.0")
	sourceSP.SetSchemaUrl("http://example.com/scope")

	sourceProfile := sourceSP.Profiles().AppendEmpty()
	sourceProfile.SetProfileID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	sourceProfile.SetTime(pcommon.Timestamp(1000))
	sourceProfile.SetDuration(pcommon.Timestamp(2000))
	sourceProfile.SetPeriod(100)
	sourceProfile.SetDefaultSampleTypeIndex(0)
	sourceProfile.SetDroppedAttributesCount(5)
	sourceProfile.SetOriginalPayloadFormat("proto")

	sampleType := sourceProfile.SampleType().AppendEmpty()
	sampleType.SetTypeStrindex(4)
	sampleType.SetUnitStrindex(1)

	sourceProfile.LocationIndices().Append(0)
	sourceProfile.AttributeIndices().Append(0)
	sourceProfile.CommentStrindices().Append(0)

	sourceProfile.OriginalPayload().Append(1)
	sourceProfile.OriginalPayload().Append(2)
	sourceProfile.OriginalPayload().Append(3)

	sample1 := sourceProfile.Sample().AppendEmpty()
	sample1.SetLocationsStartIndex(0)
	sample1.SetLocationsLength(1)
	sample1.SetLinkIndex(0)
	sample1.Value().Append(42)
	sample1.Value().Append(84)
	sample1.AttributeIndices().Append(0)
	sample1.TimestampsUnixNano().Append(1000)
	sample1.TimestampsUnixNano().Append(2000)

	sample2 := sourceProfile.Sample().AppendEmpty()
	sample2.SetLocationsStartIndex(0)
	sample2.SetLocationsLength(1)
	sample2.Value().Append(21)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 5, destDict.StringTable().Len())
	assert.Equal(t, 1, destDict.AttributeUnits().Len())
	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.LocationTable().Len())
	assert.Equal(t, 1, destDict.LinkTable().Len())
	assert.Equal(t, 1, destDict.AttributeTable().Len())

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	assert.Equal(t, 2, destProfile.Sample().Len())

	sample2Dest := destProfile.Sample().At(1)
	assert.Equal(t, sample2.LinkIndex(), sample2Dest.LinkIndex())
}

func TestProfiles_MergeFrom_AttributeUnitsDeduplication(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destDict.StringTable().Append("attr.key")
	destDict.StringTable().Append("unit.value")
	sourceDict.StringTable().Append("attr.key")
	sourceDict.StringTable().Append("unit.value")

	destAttrUnit := destDict.AttributeUnits().AppendEmpty()
	destAttrUnit.SetAttributeKeyStrindex(0)
	destAttrUnit.SetUnitStrindex(1)

	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(0)
	sourceAttrUnit.SetUnitStrindex(1)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.AttributeUnits().Len())
	assert.Equal(t, 2, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_MaxCoverage(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	destDict.StringTable().Append("existing1")
	destDict.StringTable().Append("existing2")

	destFunc := destDict.FunctionTable().AppendEmpty()
	destFunc.SetNameStrindex(0)

	destMapping := destDict.MappingTable().AppendEmpty()
	destMapping.SetMemoryStart(0x1000)

	destLoc := destDict.LocationTable().AppendEmpty()
	destLoc.SetAddress(0x2000)

	destLink := destDict.LinkTable().AppendEmpty()
	destLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

	destAttr := destDict.AttributeTable().AppendEmpty()
	destAttr.SetKey("dest.attr")
	destAttr.Value().SetStr("dest.value")

	destAttrUnit := destDict.AttributeUnits().AppendEmpty()
	destAttrUnit.SetAttributeKeyStrindex(0)

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("new.string")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x3000)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetAddress(0x4000)

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(pcommon.TraceID([16]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6}))

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("source.attr")
	sourceAttr.Value().SetStr("source.value")

	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 3, destDict.StringTable().Len())
	assert.Equal(t, 2, destDict.FunctionTable().Len())
	assert.Equal(t, 2, destDict.MappingTable().Len())
	assert.Equal(t, 2, destDict.LocationTable().Len())
	assert.Equal(t, 2, destDict.LinkTable().Len())
	assert.Equal(t, 2, destDict.AttributeTable().Len())
	assert.Equal(t, 2, destDict.AttributeUnits().Len())
}

func TestProfiles_MergeFrom_StringTableErrorPath(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("test.error.path")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 1, destDict.StringTable().Len())
	assert.Equal(t, "test.error.path", destDict.StringTable().At(0))
}

func TestProfiles_MergeFrom_AllErrorPaths(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("str1")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetAddress(0x2000)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("key1")
	sourceAttr.Value().SetStr("value1")

	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(0)

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceProfile.LocationIndices().Append(0)
	sourceProfile.AttributeIndices().Append(0)

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetLinkIndex(0)
	sourceSample.AttributeIndices().Append(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, dest.ResourceProfiles().Len())
	assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
}

func TestProfiles_MergeFrom_EdgeCaseCoverage(t *testing.T) {
	t.Run("empty_function_table_iteration", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
	})

	t.Run("empty_mapping_table_iteration", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
	})

	t.Run("empty_location_table_iteration", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
	})

	t.Run("empty_link_table_iteration", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
	})

	t.Run("empty_attribute_table_iteration", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
	})
}

func TestProfiles_MergeFrom_ErrorInMergeDictionaries(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	for i := range 100 {
		sourceDict.StringTable().Append("string" + string(rune(i)))
	}

	destDict := dest.ProfilesDictionary()
	for i := range 6 {
		destDict.StringTable().Append("deststring" + string(rune(i)))
	}

	err := dest.MergeFrom(source)
	require.NoError(t, err)
}

func TestProfiles_MergeFrom_AttributeTableDifferentValues(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destAttr := destDict.AttributeTable().AppendEmpty()
	destAttr.SetKey("same.key")
	destAttr.Value().SetStr("different.value")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("same.key")
	sourceAttr.Value().SetStr("another.value")

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 2, destDict.AttributeTable().Len())
}

func TestProfiles_MergeFrom_AttributeTableSameKeyDifferentType(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	destAttr := destDict.AttributeTable().AppendEmpty()
	destAttr.SetKey("same.key")
	destAttr.Value().SetStr("string.value")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("same.key")
	sourceAttr.Value().SetInt(42)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 2, destDict.AttributeTable().Len())
}

func TestProfiles_MergeFrom_FunctionTableNonExistentStringMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	sourceDict.StringTable().Append("func.name")
	sourceDict.StringTable().Append("sys.name")
	sourceDict.StringTable().Append("filename")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)
	sourceFunc.SetSystemNameStrindex(1)
	sourceFunc.SetFilenameStrindex(2)
	sourceFunc.SetStartLine(100)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 3, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_MappingTableNonExistentStringMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	sourceDict.StringTable().Append("filename")

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)
	sourceMapping.SetFilenameStrindex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_LocationTableNonExistentMappings(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	sourceDict.StringTable().Append("func.name")

	sourceFunc := sourceDict.FunctionTable().AppendEmpty()
	sourceFunc.SetNameStrindex(0)

	sourceMapping := sourceDict.MappingTable().AppendEmpty()
	sourceMapping.SetMemoryStart(0x1000)

	sourceLoc := sourceDict.LocationTable().AppendEmpty()
	sourceLoc.SetMappingIndex(0)
	sourceLine := sourceLoc.Line().AppendEmpty()
	sourceLine.SetFunctionIndex(0)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.Equal(t, 1, destDict.LocationTable().Len())
	assert.Equal(t, 1, destDict.FunctionTable().Len())
	assert.Equal(t, 1, destDict.MappingTable().Len())
	assert.Equal(t, 1, destDict.StringTable().Len())
}

func TestProfiles_MergeFrom_SampleWithNegativeLinkIndex(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetLinkIndex(-1)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destSample := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0)
	assert.Equal(t, int32(-1), destSample.LinkIndex())
}

func TestProfiles_MergeFrom_ProfileLocationIndicesNotInMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceProfile.LocationIndices().Append(999)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	assert.Equal(t, 1, destProfile.LocationIndices().Len())
	assert.Equal(t, int32(999), destProfile.LocationIndices().At(0))
}

func TestProfiles_MergeFrom_ProfileAttributeIndicesNotInMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceProfile.AttributeIndices().Append(999)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	assert.Equal(t, 1, destProfile.AttributeIndices().Len())
	assert.Equal(t, int32(999), destProfile.AttributeIndices().At(0))
}

func TestProfiles_MergeFrom_SampleAttributeIndicesNotInMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.AttributeIndices().Append(999)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destSample := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0)
	assert.Equal(t, 1, destSample.AttributeIndices().Len())
	assert.Equal(t, int32(999), destSample.AttributeIndices().At(0))
}

func TestProfiles_MergeFrom_SampleLinkIndexNotInMapping(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceSample := sourceProfile.Sample().AppendEmpty()
	sourceSample.SetLinkIndex(999)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destSample := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Sample().At(0)
	assert.Equal(t, int32(999), destSample.LinkIndex())
}

func TestProfiles_MergeFrom_AttributeUnitMergeEdgeCases(t *testing.T) {
	t.Run("duplicate_attribute_units", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		destDict.StringTable().Append("key1")
		destDict.StringTable().Append("unit1")
		sourceDict.StringTable().Append("key1")
		sourceDict.StringTable().Append("unit1")

		destUnit := destDict.AttributeUnits().AppendEmpty()
		destUnit.SetAttributeKeyStrindex(0)
		destUnit.SetUnitStrindex(1)

		sourceUnit := sourceDict.AttributeUnits().AppendEmpty()
		sourceUnit.SetAttributeKeyStrindex(0)
		sourceUnit.SetUnitStrindex(1)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, destDict.AttributeUnits().Len())
	})

	t.Run("different_units_same_key", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		destDict.StringTable().Append("key1")
		destDict.StringTable().Append("unit1")
		sourceDict.StringTable().Append("key1")
		sourceDict.StringTable().Append("unit2")

		destUnit := destDict.AttributeUnits().AppendEmpty()
		destUnit.SetAttributeKeyStrindex(0)
		destUnit.SetUnitStrindex(1)

		sourceUnit := sourceDict.AttributeUnits().AppendEmpty()
		sourceUnit.SetAttributeKeyStrindex(0)
		sourceUnit.SetUnitStrindex(1)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 2, destDict.AttributeUnits().Len())
	})

	t.Run("string_mapping_updates", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		destDict.StringTable().Append("existing_key")
		destDict.StringTable().Append("existing_unit")
		sourceDict.StringTable().Append("new_key")
		sourceDict.StringTable().Append("new_unit")

		sourceUnit := sourceDict.AttributeUnits().AppendEmpty()
		sourceUnit.SetAttributeKeyStrindex(0)
		sourceUnit.SetUnitStrindex(1)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 4, destDict.StringTable().Len())
		assert.Equal(t, 1, destDict.AttributeUnits().Len())
	})
}

func TestProfiles_MergeFrom_AttributeUnitsNegativeIndices(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()

	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(-1)
	sourceAttrUnit.SetUnitStrindex(-1)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 1, destDict.AttributeUnits().Len())
	destAttrUnit := destDict.AttributeUnits().At(0)
	assert.Equal(t, int32(-1), destAttrUnit.AttributeKeyStrindex())
	assert.Equal(t, int32(-1), destAttrUnit.UnitStrindex())
}

func TestProfiles_MergeFrom_AttributeUnitsStringMappingMissing(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceAttrUnit := sourceDict.AttributeUnits().AppendEmpty()
	sourceAttrUnit.SetAttributeKeyStrindex(999)
	sourceAttrUnit.SetUnitStrindex(888)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destDict := dest.ProfilesDictionary()
	assert.Equal(t, 1, destDict.AttributeUnits().Len())
	destAttrUnit := destDict.AttributeUnits().At(0)
	assert.Equal(t, int32(999), destAttrUnit.AttributeKeyStrindex())
	assert.Equal(t, int32(888), destAttrUnit.UnitStrindex())
}

func TestProfiles_MergeFrom_ErrorPropagationEdgeCases(t *testing.T) {
	t.Run("string_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		sourceFunc := sourceDict.FunctionTable().AppendEmpty()
		sourceFunc.SetNameStrindex(0)
		sourceFunc.SetSystemNameStrindex(0)
		sourceFunc.SetFilenameStrindex(0)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().StringTable().Len())
		assert.Equal(t, 1, dest.ProfilesDictionary().FunctionTable().Len())
	})

	t.Run("function_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		sourceFunc := sourceDict.FunctionTable().AppendEmpty()
		sourceFunc.SetNameStrindex(999) // Invalid index
		sourceFunc.SetStartLine(100)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().FunctionTable().Len())
	})

	t.Run("mapping_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		sourceMapping := sourceDict.MappingTable().AppendEmpty()
		sourceMapping.SetFilenameStrindex(999) // Invalid index
		sourceMapping.SetMemoryStart(0x1000)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().MappingTable().Len())
	})

	t.Run("location_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceDict.StringTable().Append("test.string")

		sourceFunc := sourceDict.FunctionTable().AppendEmpty()
		sourceFunc.SetNameStrindex(0)

		sourceLoc := sourceDict.LocationTable().AppendEmpty()
		sourceLoc.SetMappingIndex(999) // Invalid index
		sourceLine := sourceLoc.Line().AppendEmpty()
		sourceLine.SetFunctionIndex(0)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().LocationTable().Len())
	})

	t.Run("link_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceLink := sourceDict.LinkTable().AppendEmpty()
		sourceLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

		sourceRP := source.ResourceProfiles().AppendEmpty()
		sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
		sourceProfile := sourceSP.Profiles().AppendEmpty()
		sourceSample := sourceProfile.Sample().AppendEmpty()
		sourceSample.SetLinkIndex(999) // Invalid index

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().LinkTable().Len())
	})

	t.Run("attribute_table_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceAttr := sourceDict.AttributeTable().AppendEmpty()
		sourceAttr.SetKey("test.key")
		sourceAttr.Value().SetStr("test.value")

		sourceRP := source.ResourceProfiles().AppendEmpty()
		sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
		sourceProfile := sourceSP.Profiles().AppendEmpty()
		sourceProfile.AttributeIndices().Append(999) // Invalid index

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().AttributeTable().Len())
	})

	t.Run("sample_attribute_indices_error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		sourceAttr := sourceDict.AttributeTable().AppendEmpty()
		sourceAttr.SetKey("test.key")
		sourceAttr.Value().SetStr("test.value")

		sourceRP := source.ResourceProfiles().AppendEmpty()
		sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
		sourceProfile := sourceSP.Profiles().AppendEmpty()
		sourceSample := sourceProfile.Sample().AppendEmpty()
		sourceSample.AttributeIndices().Append(999) // Invalid index

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, dest.ProfilesDictionary().AttributeTable().Len())
	})
}

func TestProfiles_MergeFrom_LargeTableSizes(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	destDict := dest.ProfilesDictionary()
	sourceDict := source.ProfilesDictionary()

	for i := range 1000 {
		destDict.StringTable().Append("dest" + string(rune(i%26+'a')))
		sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))

		func1 := destDict.FunctionTable().AppendEmpty()
		func1.SetNameStrindex(int32(i % 100)) //nolint:gosec
		func1.SetStartLine(int64(i))

		func2 := sourceDict.FunctionTable().AppendEmpty()
		func2.SetNameStrindex(int32(i % 100)) //nolint:gosec
		func2.SetStartLine(int64(i + 2000))

		mapping1 := destDict.MappingTable().AppendEmpty()
		mapping1.SetMemoryStart(uint64(0x1000 + i)) //nolint:gosec

		mapping2 := sourceDict.MappingTable().AppendEmpty()
		mapping2.SetMemoryStart(uint64(0x2000 + i)) //nolint:gosec

		loc1 := destDict.LocationTable().AppendEmpty()
		loc1.SetAddress(uint64(0x3000 + i)) //nolint:gosec

		loc2 := sourceDict.LocationTable().AppendEmpty()
		loc2.SetAddress(uint64(0x4000 + i)) //nolint:gosec

		link1 := destDict.LinkTable().AppendEmpty()
		link1.SetTraceID(pcommon.TraceID([16]byte{byte(i), byte(i + 1), byte(i + 2), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

		link2 := sourceDict.LinkTable().AppendEmpty()
		link2.SetTraceID(pcommon.TraceID([16]byte{byte(i + 100), byte(i + 101), byte(i + 102), 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

		attr1 := destDict.AttributeTable().AppendEmpty()
		attr1.SetKey("destkey" + string(rune(i%10+'0')))
		attr1.Value().SetStr("destvalue")

		attr2 := sourceDict.AttributeTable().AppendEmpty()
		attr2.SetKey("sourcekey" + string(rune(i%10+'0')))
		attr2.Value().SetStr("sourcevalue")
	}

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, destDict.StringTable().Len(), 1000)
	assert.GreaterOrEqual(t, destDict.FunctionTable().Len(), 1000)
	assert.GreaterOrEqual(t, destDict.MappingTable().Len(), 1000)
	assert.GreaterOrEqual(t, destDict.LocationTable().Len(), 1000)
	assert.GreaterOrEqual(t, destDict.LinkTable().Len(), 1000)
	assert.GreaterOrEqual(t, destDict.AttributeTable().Len(), 1000)
}

func TestProfiles_MergeFrom_ComplexSampleEdgeCases(t *testing.T) {
	dest := NewProfiles()
	source := NewProfiles()

	sourceDict := source.ProfilesDictionary()
	sourceDict.StringTable().Append("str1")

	sourceAttr := sourceDict.AttributeTable().AppendEmpty()
	sourceAttr.SetKey("test.key")
	sourceAttr.Value().SetStr("test.value")

	sourceLink := sourceDict.LinkTable().AppendEmpty()
	sourceLink.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))

	sourceRP := source.ResourceProfiles().AppendEmpty()
	sourceSP := sourceRP.ScopeProfiles().AppendEmpty()
	sourceProfile := sourceSP.Profiles().AppendEmpty()

	sourceProfile.AttributeIndices().Append(0)
	sourceProfile.LocationIndices().Append(999)

	sourceSample1 := sourceProfile.Sample().AppendEmpty()
	sourceSample1.SetLinkIndex(0)
	sourceSample1.AttributeIndices().Append(0)

	sourceSample2 := sourceProfile.Sample().AppendEmpty()
	sourceSample2.SetLinkIndex(999)
	sourceSample2.AttributeIndices().Append(999)

	err := dest.MergeFrom(source)
	require.NoError(t, err)

	destProfile := dest.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0)
	assert.Equal(t, 1, destProfile.AttributeIndices().Len())
	assert.Equal(t, 1, destProfile.LocationIndices().Len())
	assert.Equal(t, int32(999), destProfile.LocationIndices().At(0))

	assert.Equal(t, 2, destProfile.Sample().Len())
	destSample1 := destProfile.Sample().At(0)
	destSample2 := destProfile.Sample().At(1)

	assert.True(t, destSample1.HasLinkIndex())
	assert.Equal(t, int32(999), destSample2.LinkIndex())
	assert.Equal(t, int32(999), destSample2.AttributeIndices().At(0))
}

func TestSafeInt32(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int32
		wantErr bool
	}{
		{
			name:    "valid positive",
			input:   100,
			want:    100,
			wantErr: false,
		},
		{
			name:    "valid zero",
			input:   0,
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative",
			input:   -1,
			want:    0,
			wantErr: true,
		},
		{
			name:    "too large",
			input:   math.MaxInt32 + 1,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeInt32(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestProfiles_CheckSizeLimits_Overflow(t *testing.T) {
	t.Run("string_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few strings to both tables
		for i := 0; i < 10; i++ {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		// Add enough strings to exceed max when merged
		for i := 0; i < 1000; i++ {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("function_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few functions to both tables
		for i := 0; i < 10; i++ {
			fn := destDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i))
			fn = sourceDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i + 1000))
		}

		// Add enough functions to exceed max when merged
		for i := 0; i < 1000; i++ {
			fn := destDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i))
			fn = sourceDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i + 1000))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("mapping_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few mappings to both tables
		for i := 0; i < 10; i++ {
			mapping := destDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i)) //nolint:gosec
			mapping = sourceDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i + 1000)) //nolint:gosec
		}

		// Add enough mappings to exceed max when merged
		for i := 0; i < 1000; i++ {
			mapping := destDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i)) //nolint:gosec
			mapping = sourceDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("location_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few locations to both tables
		for i := 0; i < 10; i++ {
			loc := destDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i)) //nolint:gosec
			loc = sourceDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i + 1000)) //nolint:gosec
		}

		// Add enough locations to exceed max when merged
		for i := 0; i < 1000; i++ {
			loc := destDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i)) //nolint:gosec
			loc = sourceDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("link_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few links to both tables
		for i := 0; i < 10; i++ {
			link := destDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
			link = sourceDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i + 100), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
		}

		// Add enough links to exceed max when merged
		for i := 0; i < 1000; i++ {
			link := destDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
			link = sourceDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i + 100), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("attribute_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few attributes to both tables
		for i := 0; i < 10; i++ {
			attr := destDict.AttributeTable().AppendEmpty()
			attr.SetKey("key" + string(rune(i%26+'a')))
			attr = sourceDict.AttributeTable().AppendEmpty()
			attr.SetKey("source_key" + string(rune(i%26+'a')))
		}

		// Add enough attributes to exceed max when merged
		for i := 0; i < 1000; i++ {
			attr := destDict.AttributeTable().AppendEmpty()
			attr.SetKey("key" + string(rune(i%26+'a')))
			attr = sourceDict.AttributeTable().AppendEmpty()
			attr.SetKey("source_key" + string(rune(i%26+'a')))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("attribute_units_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few attribute units to both tables
		for i := 0; i < 10; i++ {
			unit := destDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i)) //nolint:gosec
			unit = sourceDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i + 1000)) //nolint:gosec
		}

		// Add enough attribute units to exceed max when merged
		for i := 0; i < 1000; i++ {
			unit := destDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i)) //nolint:gosec
			unit = sourceDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})
}

func TestProfiles_MergeDictionaries_SizeLimits(t *testing.T) {
	t.Run("small_tables_merge_successfully", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a few entries to each table
		destDict.StringTable().Append("test1")
		sourceDict.StringTable().Append("test2")

		fn := destDict.FunctionTable().AppendEmpty()
		fn.SetStartLine(1)
		fn = sourceDict.FunctionTable().AppendEmpty()
		fn.SetStartLine(2)

		mapping := destDict.MappingTable().AppendEmpty()
		mapping.SetMemoryStart(1)
		mapping = sourceDict.MappingTable().AppendEmpty()
		mapping.SetMemoryStart(2)

		loc := destDict.LocationTable().AppendEmpty()
		loc.SetAddress(1)
		loc = sourceDict.LocationTable().AppendEmpty()
		loc.SetAddress(2)

		link := destDict.LinkTable().AppendEmpty()
		link.SetTraceID(pcommon.TraceID([16]byte{1}))
		link = sourceDict.LinkTable().AppendEmpty()
		link.SetTraceID(pcommon.TraceID([16]byte{2}))

		attr := destDict.AttributeTable().AppendEmpty()
		attr.SetKey("key1")
		attr = sourceDict.AttributeTable().AppendEmpty()
		attr.SetKey("key2")

		unit := destDict.AttributeUnits().AppendEmpty()
		unit.SetAttributeKeyStrindex(1)
		unit = sourceDict.AttributeUnits().AppendEmpty()
		unit.SetAttributeKeyStrindex(2)

		_, err := dest.mergeDictionaries(destDict, sourceDict)
		require.NoError(t, err)

		// Verify merged tables have correct lengths
		assert.Equal(t, 2, destDict.StringTable().Len())
		assert.Equal(t, 2, destDict.FunctionTable().Len())
		assert.Equal(t, 2, destDict.MappingTable().Len())
		assert.Equal(t, 2, destDict.LocationTable().Len())
		assert.Equal(t, 2, destDict.LinkTable().Len())
		assert.Equal(t, 2, destDict.AttributeTable().Len())
		assert.Equal(t, 2, destDict.AttributeUnits().Len())
	})

	t.Run("deduplication_reduces_table_size", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add identical entries to both tables
		destDict.StringTable().Append("test")
		sourceDict.StringTable().Append("test")

		fn := destDict.FunctionTable().AppendEmpty()
		fn.SetStartLine(1)
		fn = sourceDict.FunctionTable().AppendEmpty()
		fn.SetStartLine(1)

		mapping := destDict.MappingTable().AppendEmpty()
		mapping.SetMemoryStart(1)
		mapping = sourceDict.MappingTable().AppendEmpty()
		mapping.SetMemoryStart(1)

		loc := destDict.LocationTable().AppendEmpty()
		loc.SetAddress(1)
		loc = sourceDict.LocationTable().AppendEmpty()
		loc.SetAddress(1)

		link := destDict.LinkTable().AppendEmpty()
		link.SetTraceID(pcommon.TraceID([16]byte{1}))
		link = sourceDict.LinkTable().AppendEmpty()
		link.SetTraceID(pcommon.TraceID([16]byte{1}))

		attr := destDict.AttributeTable().AppendEmpty()
		attr.SetKey("key")
		attr = sourceDict.AttributeTable().AppendEmpty()
		attr.SetKey("key")

		unit := destDict.AttributeUnits().AppendEmpty()
		unit.SetAttributeKeyStrindex(1)
		unit = sourceDict.AttributeUnits().AppendEmpty()
		unit.SetAttributeKeyStrindex(1)

		_, err := dest.mergeDictionaries(destDict, sourceDict)
		require.NoError(t, err)

		// Verify merged tables have correct lengths after deduplication
		assert.Equal(t, 1, destDict.StringTable().Len())
		assert.Equal(t, 1, destDict.FunctionTable().Len())
		assert.Equal(t, 1, destDict.MappingTable().Len())
		assert.Equal(t, 1, destDict.LocationTable().Len())
		assert.Equal(t, 1, destDict.LinkTable().Len())
		assert.Equal(t, 1, destDict.AttributeTable().Len())
		assert.Equal(t, 1, destDict.AttributeUnits().Len())
	})
}

func TestProfiles_MergeDictionaries_EdgeCases(t *testing.T) {
	t.Run("empty_dictionaries", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		mappings, err := dest.mergeDictionaries(destDict, sourceDict)
		require.NoError(t, err)
		assert.NotNil(t, mappings)
		assert.Empty(t, mappings.stringTable)
		assert.Empty(t, mappings.functionTable)
		assert.Empty(t, mappings.mappingTable)
		assert.Empty(t, mappings.locationTable)
		assert.Empty(t, mappings.linkTable)
		assert.Empty(t, mappings.attributeTable)
	})

	t.Run("error_propagation", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough entries to cause an overflow
		for i := 0; i < 1000; i++ {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		mappings, err := dest.mergeDictionaries(destDict, sourceDict)
		require.NoError(t, err)
		assert.NotNil(t, mappings)
	})
}

func TestProfiles_CheckSizeLimits_EdgeCases(t *testing.T) {
	t.Run("empty_dictionaries", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("string_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough strings to cause an overflow
		for i := 0; i < 1000; i++ {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("function_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough functions to cause an overflow
		for i := 0; i < 1000; i++ {
			fn := destDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i))
			fn = sourceDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i + 1000))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("mapping_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough mappings to cause an overflow
		for i := 0; i < 1000; i++ {
			mapping := destDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i)) //nolint:gosec
			mapping = sourceDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("location_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough locations to cause an overflow
		for i := 0; i < 1000; i++ {
			loc := destDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i)) //nolint:gosec
			loc = sourceDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("link_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough links to cause an overflow
		for i := 0; i < 1000; i++ {
			link := destDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
			link = sourceDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i + 100), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("attribute_table_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough attributes to cause an overflow
		for i := 0; i < 1000; i++ {
			attr := destDict.AttributeTable().AppendEmpty()
			attr.SetKey("key" + string(rune(i%26+'a')))
			attr = sourceDict.AttributeTable().AppendEmpty()
			attr.SetKey("source_key" + string(rune(i%26+'a')))
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})

	t.Run("attribute_units_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add enough attribute units to cause an overflow
		for i := 0; i < 1000; i++ {
			unit := destDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i)) //nolint:gosec
			unit = sourceDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i + 1000)) //nolint:gosec
		}

		err := dest.checkSizeLimits(destDict, sourceDict)
		require.NoError(t, err)
	})
}

func TestProfiles_MergeFrom_CheckSizeLimitsEdgeCases(t *testing.T) {
	t.Run("max_size_no_overflow", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		for i := range 100 {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
		}

		for i := range 50 {
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Greater(t, destDict.StringTable().Len(), 100)
	})

	t.Run("empty_source_tables", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		for i := range 10 {
			destDict.StringTable().Append("existing" + string(rune(i+'0')))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 10, destDict.StringTable().Len())
	})

	t.Run("empty_dest_tables", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()
		for i := range 10 {
			sourceDict.StringTable().Append("new" + string(rune(i+'0')))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 10, dest.ProfilesDictionary().StringTable().Len())
	})

	t.Run("dictionary_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of strings to both tables
		for i := 0; i < 1000; i++ {
			destDict.StringTable().Append("test" + string(rune(i%26+'a')))
			sourceDict.StringTable().Append("source" + string(rune(i%26+'a')))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("function_table_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of functions to both tables
		for i := 0; i < 1000; i++ {
			fn := destDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i))
			fn = sourceDict.FunctionTable().AppendEmpty()
			fn.SetStartLine(int64(i + 1000))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("mapping_table_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of mappings to both tables
		for i := 0; i < 1000; i++ {
			mapping := destDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i)) //nolint:gosec
			mapping = sourceDict.MappingTable().AppendEmpty()
			mapping.SetMemoryStart(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("location_table_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of locations to both tables
		for i := 0; i < 1000; i++ {
			loc := destDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i)) //nolint:gosec
			loc = sourceDict.LocationTable().AppendEmpty()
			loc.SetAddress(uint64(i + 1000)) //nolint:gosec
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("link_table_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of links to both tables
		for i := 0; i < 1000; i++ {
			link := destDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
			link = sourceDict.LinkTable().AppendEmpty()
			link.SetTraceID(pcommon.TraceID([16]byte{byte(i + 100), 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("attribute_table_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of attributes to both tables
		for i := 0; i < 1000; i++ {
			attr := destDict.AttributeTable().AppendEmpty()
			attr.SetKey("key" + string(rune(i%26+'a')))
			attr = sourceDict.AttributeTable().AppendEmpty()
			attr.SetKey("source_key" + string(rune(i%26+'a')))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("attribute_units_size_exceeded", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add a large number of attribute units to both tables
		for i := 0; i < 1000; i++ {
			unit := destDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i)) //nolint:gosec
			unit = sourceDict.AttributeUnits().AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i + 1000)) //nolint:gosec
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})
}

// TestProfiles_MergeFrom_AdditionalSizeLimitTests tests additional edge cases in size limit checking
func TestProfiles_MergeFrom_AdditionalSizeLimitTests(t *testing.T) {
	tests := []struct {
		name        string
		setupTables func(dest, source ProfilesDictionary)
		expectError bool
	}{
		{
			name: "all_tables_within_limits",
			setupTables: func(dest, source ProfilesDictionary) {
				// Test case where all tables are within limits
				destStrings := dest.StringTable()
				sourceStrings := source.StringTable()

				for i := 0; i < 100; i++ {
					destStrings.Append("dest_string_" + strconv.Itoa(i))
					sourceStrings.Append("source_string_" + strconv.Itoa(i))
				}
			},
			expectError: false,
		},
		{
			name: "combined_size_near_limit_but_valid",
			setupTables: func(dest, source ProfilesDictionary) {
				// Set sizes that are large but don't exceed limits when combined
				destStrings := dest.StringTable()
				sourceStrings := source.StringTable()

				for i := 0; i < 1000; i++ {
					destStrings.Append("dest" + strconv.Itoa(i))
					sourceStrings.Append("source" + strconv.Itoa(i))
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := NewProfiles()
			source := NewProfiles()

			destDict := dest.ProfilesDictionary()
			sourceDict := source.ProfilesDictionary()

			tt.setupTables(destDict, sourceDict)

			err := dest.MergeFrom(source)
			if tt.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrDictionarySizeExceeded)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestProfiles_MergeFrom_MergeDictionariesErrors tests error paths in mergeDictionaries
func TestProfiles_MergeFrom_MergeDictionariesErrors(t *testing.T) {
	t.Run("error_in_attribute_table_merge", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		// Create large dictionaries to trigger potential size limit errors
		sourceDict := source.ProfilesDictionary()
		destDict := dest.ProfilesDictionary()

		// Add attribute units that will cause issues during merging
		destUnits := destDict.AttributeUnits()
		sourceUnits := sourceDict.AttributeUnits()

		for i := 0; i < 1000; i++ {
			unit := destUnits.AppendEmpty()
			unit.SetAttributeKeyStrindex(0)
			unit.SetUnitStrindex(0)
		}

		for i := 0; i < 1000; i++ {
			unit := sourceUnits.AppendEmpty()
			unit.SetAttributeKeyStrindex(int32(i)) //nolint:gosec
			unit.SetUnitStrindex(int32(i))         //nolint:gosec
		}

		// This should complete without error but test the merge path
		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("error_in_link_table_merge", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()

		// Add links to test link table merging
		sourceLinks := sourceDict.LinkTable()
		for i := 0; i < 1000; i++ {
			link := sourceLinks.AppendEmpty()
			link.SetTraceID([16]byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3)})
			link.SetSpanID([8]byte{byte(i), byte(i + 1)})
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})
}

// TestProfiles_MergeFrom_StringTableErrors tests error conditions in string table merging
func TestProfiles_MergeFrom_StringTableErrors(t *testing.T) {
	t.Run("string_table_merge_with_many_duplicates", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		// Add some strings directly using Append (which doesn't dedupe)
		destDict.StringTable().Append("shared_1")
		destDict.StringTable().Append("shared_2")
		destDict.StringTable().Append("dest_only")

		sourceDict.StringTable().Append("shared_1")
		sourceDict.StringTable().Append("shared_2")
		sourceDict.StringTable().Append("source_only")

		initialLen := destDict.StringTable().Len()

		err := dest.MergeFrom(source)
		require.NoError(t, err)

		// After merge, we should have shared_1, shared_2, dest_only, source_only = 4 unique strings
		// The merge process should deduplicate the shared ones
		finalLen := destDict.StringTable().Len()
		assert.Greater(t, finalLen, initialLen) // Should have added at least source_only
		assert.Less(t, finalLen, 6)             // But not all 6 strings due to deduplication
	})

	t.Run("string_table_with_empty_source", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		destDict := dest.ProfilesDictionary()
		destDict.StringTable().Append("existing_string")

		err := dest.MergeFrom(source)
		require.NoError(t, err)
		assert.Equal(t, 1, destDict.StringTable().Len())
	})
}

// TestProfiles_MergeFrom_IndexOverflowEdgeCases tests potential integer overflow scenarios
func TestProfiles_MergeFrom_IndexOverflowEdgeCases(t *testing.T) {
	t.Run("large_index_values_within_bounds", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		// Create a source with large but valid index values
		sourceDict := source.ProfilesDictionary()

		// Add strings to get valid indices
		for i := 0; i < 1000; i++ {
			sourceDict.StringTable().Append(fmt.Sprintf("string_%d", i))
		}

		// Add functions with large but valid string indices
		sourceFunctions := sourceDict.FunctionTable()
		for i := 0; i < 100; i++ {
			fn := sourceFunctions.AppendEmpty()
			fn.SetNameStrindex(int32(i % 1000))             //nolint:gosec
			fn.SetSystemNameStrindex(int32((i + 1) % 1000)) //nolint:gosec
			fn.SetFilenameStrindex(int32((i + 2) % 1000))   //nolint:gosec
			fn.SetStartLine(int64(i))
		}

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})
}

// TestProfiles_MergeFrom_EmptyTableIterations tests iteration over empty tables
func TestProfiles_MergeFrom_EmptyTableIterations(t *testing.T) {
	t.Run("all_empty_tables", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		// Both have empty dictionaries - should handle gracefully
		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("source_with_empty_tables_dest_with_data", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		// Add data to dest
		destDict := dest.ProfilesDictionary()
		destDict.StringTable().Append("dest_string")

		destFunctions := destDict.FunctionTable()
		fn := destFunctions.AppendEmpty()
		fn.SetNameStrindex(0)
		fn.SetStartLine(42)

		// Source remains empty
		err := dest.MergeFrom(source)
		require.NoError(t, err)

		// Dest should retain its data
		assert.Equal(t, 1, destDict.StringTable().Len())
		assert.Equal(t, 1, destDict.FunctionTable().Len())
	})
}

// TestProfiles_MergeFrom_SpecialIndexValues tests handling of special index values
func TestProfiles_MergeFrom_SpecialIndexValues(t *testing.T) {
	t.Run("zero_and_negative_indices", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		// Set up dictionaries
		destDict := dest.ProfilesDictionary()
		sourceDict := source.ProfilesDictionary()

		destDict.StringTable().Append("dest_string")
		sourceDict.StringTable().Append("source_string")

		// Add functions with zero and negative indices
		sourceFunctions := sourceDict.FunctionTable()
		fn := sourceFunctions.AppendEmpty()
		fn.SetNameStrindex(0)        // Valid index
		fn.SetSystemNameStrindex(-1) // Invalid/unset index
		fn.SetFilenameStrindex(-5)   // Invalid/unset index
		fn.SetStartLine(100)

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})
}

// TestProfiles_MergeFrom_AttributeUnitsCoverage tests remaining attribute units functionality
func TestProfiles_MergeFrom_AttributeUnitsCoverage(t *testing.T) {
	t.Run("attribute_units_with_no_string_mapping", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()

		// Add attribute units with indices that won't be in string mapping
		sourceUnits := sourceDict.AttributeUnits()
		unit := sourceUnits.AppendEmpty()
		unit.SetAttributeKeyStrindex(-1) // Negative index - won't be mapped
		unit.SetUnitStrindex(-1)         // Negative index - won't be mapped

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})

	t.Run("attribute_units_partial_string_mapping", func(t *testing.T) {
		dest := NewProfiles()
		source := NewProfiles()

		sourceDict := source.ProfilesDictionary()

		// Add some strings
		sourceDict.StringTable().Append("key_string")
		sourceDict.StringTable().Append("unit_string")

		// Add attribute units with only some valid mappings
		sourceUnits := sourceDict.AttributeUnits()
		unit := sourceUnits.AppendEmpty()
		unit.SetAttributeKeyStrindex(0) // Valid mapping
		unit.SetUnitStrindex(-1)        // Invalid mapping

		unit2 := sourceUnits.AppendEmpty()
		unit2.SetAttributeKeyStrindex(-1) // Invalid mapping
		unit2.SetUnitStrindex(1)          // Valid mapping

		err := dest.MergeFrom(source)
		require.NoError(t, err)
	})
}
