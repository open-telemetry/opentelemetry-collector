// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var (
	_ Marshaler   = (*JSONMarshaler)(nil)
	_ Unmarshaler = (*JSONUnmarshaler)(nil)
)

var profilesOTLP = func() Profiles {
	startTimestamp := pcommon.Timestamp(1684617382541971000)
	durationTimestamp := pcommon.Timestamp(1684623646539558000)
	profileID := ProfileID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	traceID := pcommon.TraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	spanID := pcommon.SpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})

	pd := NewProfiles()
	// Add ResourceProfiles.
	rp := pd.ResourceProfiles().AppendEmpty()
	rp.SetSchemaUrl("schemaURL")
	// Add resource.
	rp.Resource().Attributes().PutStr("host.name", "testHost")
	rp.Resource().Attributes().PutStr("service.name", "testService")
	rp.Resource().SetDroppedAttributesCount(1)
	// Add ScopeProfiles.
	sp := rp.ScopeProfiles().AppendEmpty()
	sp.SetSchemaUrl("schemaURL")
	sp.Scope().SetName("scope name")
	sp.Scope().SetVersion("scope version")
	// Add profiles
	pro := sp.Profiles().AppendEmpty()
	pro.SetProfileID(profileID)
	pro.SetTime(startTimestamp)
	pro.SetDuration(durationTimestamp)
	pro.AttributeIndices().Append(1)
	pro.SetDroppedAttributesCount(1)

	// Add sample type
	st := pro.SampleType().AppendEmpty()
	st.SetTypeStrindex(1)
	st.SetUnitStrindex(2)
	// Add samples
	s := pro.Sample().AppendEmpty()
	s.SetLocationsStartIndex(1)
	s.SetLocationsLength(10)
	s.Value().Append(3)
	s.AttributeIndices().Append(1)
	s.TimestampsUnixNano().Append(12345)

	// Add mappings
	m := pro.MappingTable().AppendEmpty()
	m.SetMemoryStart(2)
	m.SetMemoryLimit(3)
	m.SetFileOffset(4)
	m.SetFilenameStrindex(5)
	m.AttributeIndices().Append(7)
	m.AttributeIndices().Append(8)
	m.SetHasFunctions(true)
	m.SetHasFilenames(true)
	m.SetHasLineNumbers(true)
	m.SetHasInlineFrames(true)

	// Add location
	l := pro.LocationTable().AppendEmpty()
	l.SetMappingIndex(3)
	l.SetAddress(4)
	l.SetIsFolded(true)
	l.AttributeIndices().Append(6)
	l.AttributeIndices().Append(7)
	li := l.Line().AppendEmpty()
	li.SetFunctionIndex(1)
	li.SetLine(2)
	li.SetColumn(3)
	pro.LocationIndices().Append(1)

	// Add function
	f := pro.FunctionTable().AppendEmpty()
	f.SetNameStrindex(2)
	f.SetSystemNameStrindex(3)
	f.SetFilenameStrindex(4)
	f.SetStartLine(5)

	// Add attribute table
	at := pro.AttributeTable()
	a := at.AppendEmpty()
	a.SetKey("answer")
	a.Value().SetInt(42)

	// Add attribute units
	au := pro.AttributeUnits().AppendEmpty()
	au.SetAttributeKeyStrindex(1)
	au.SetUnitStrindex(5)

	// Add link table
	lt := pro.LinkTable().AppendEmpty()
	lt.SetTraceID(traceID)
	lt.SetSpanID(spanID)

	// Add string table
	pro.StringTable().Append("foobar")
	pro.SetTime(1234)
	pro.SetDuration(5678)
	pro.PeriodType().SetTypeStrindex(1)
	pro.PeriodType().SetUnitStrindex(2)
	pro.SetPeriod(3)
	pro.CommentStrindices().Append(1)
	pro.CommentStrindices().Append(2)
	pro.SetDefaultSampleTypeStrindex(4)

	return pd
}()

var profilesJSON = `{"resourceProfiles":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}},{"key":"service.name","value":{"stringValue":"testService"}}],"droppedAttributesCount":1},"scopeProfiles":[{"scope":{"name":"scope name","version":"scope version"},"profiles":[{"sampleType":[{"typeStrindex":1,"unitStrindex":2}],"sample":[{"locationsStartIndex":1,"locationsLength":10,"value":["3"],"attributeIndices":[1],"timestampsUnixNano":["12345"]}],"mappingTable":[{"memoryStart":"2","memoryLimit":"3","fileOffset":"4","filenameStrindex":5,"attributeIndices":[7,8],"hasFunctions":true,"hasFilenames":true,"hasLineNumbers":true,"hasInlineFrames":true}],"locationTable":[{"mappingIndex":3,"address":"4","line":[{"functionIndex":1,"line":"2","column":"3"}],"isFolded":true,"attributeIndices":[6,7]}],"locationIndices":[1],"functionTable":[{"nameStrindex":2,"systemNameStrindex":3,"filenameStrindex":4,"startLine":"5"}],"attributeTable":[{"key":"answer","value":{"intValue":"42"}}],"attributeUnits":[{"attributeKeyStrindex":1,"unitStrindex":5}],"linkTable":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718"}],"stringTable":["foobar"],"timeNanos":"1234","durationNanos":"5678","periodType":{"typeStrindex":1,"unitStrindex":2},"period":"3","commentStrindices":[1,2],"defaultSampleTypeStrindex":4,"profileId":"0102030405060708090a0b0c0d0e0f10","attributeIndices":[1],"droppedAttributesCount":1}],"schemaUrl":"schemaURL"}],"schemaUrl":"schemaURL"}]}`

func TestJSONUnmarshal(t *testing.T) {
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalProfiles([]byte(profilesJSON))
	require.NoError(t, err)
	assert.Equal(t, profilesOTLP, got)
}

func TestJSONMarshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	require.NoError(t, err)
	assert.JSONEq(t, profilesJSON, string(jsonBuf))
}

func TestJSONUnmarshalInvalid(t *testing.T) {
	jsonStr := `{"extra":"", "resourceProfiles": "extra"}`
	decoder := &JSONUnmarshaler{}
	_, err := decoder.UnmarshalProfiles([]byte(jsonStr))
	assert.Error(t, err)
}

func TestUnmarshalJsoniterProfileData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceProfiles": [{"extra":""}]}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewProfiles()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, 1, val.ResourceProfiles().Len())
}

func TestUnmarshalJsoniterProfileInvalidProfileIDField(t *testing.T) {
	jsonStr := `{"profile_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewProfile().unmarshalJsoniter(iter)
	assert.ErrorContains(t, iter.Error, "parse profile_id")
}

func TestUnmarshalJsoniterResourceProfiles(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "scopeProfiles": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewResourceProfiles()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewResourceProfiles(), val)
}

func TestUnmarshalJsoniterScopeProfiles(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "profiles": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeProfiles()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewScopeProfiles(), val)
}

func TestUnmarshalJsoniterProfile(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewProfile()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewProfile(), val)
}

func TestUnmarshalJsoniterValueType(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewValueType()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewValueType(), val)
}

func TestUnmarshalJsoniterSample(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSample()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewSample(), val)
}

func TestUnmarshalJsoniterMapping(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewMapping()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewMapping(), val)
}

func TestUnmarshalJsoniterLocation(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLocation()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewLocation(), val)
}

func TestUnmarshalJsoniterLine(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLine()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewLine(), val)
}

func TestUnmarshalJsoniterFunction(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewFunction()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewFunction(), val)
}

func TestUnmarshalJsoniterAttributeUnit(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewAttributeUnit()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewAttributeUnit(), val)
}

func TestUnmarshalJsoniterLink(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLink()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewLink(), val)
}

func TestUnmarshalJsoniterLinkInvalidTraceIDField(t *testing.T) {
	jsonStr := `{"trace_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewLink().unmarshalJsoniter(iter)
	assert.ErrorContains(t, iter.Error, "parse trace_id")
}

func TestUnmarshalJsoniterSpanLinkInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewLink().unmarshalJsoniter(iter)
	assert.ErrorContains(t, iter.Error, "parse span_id")
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	require.NoError(b, err)
	decoder := &JSONUnmarshaler{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalProfiles(jsonBuf)
			assert.NoError(b, err)
		}
	})
}
