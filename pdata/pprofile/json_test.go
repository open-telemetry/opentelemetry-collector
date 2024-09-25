// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1experimental"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

var _ Marshaler = (*JSONMarshaler)(nil)
var _ Unmarshaler = (*JSONUnmarshaler)(nil)

var profilesOTLP = func() Profiles {
	startTimestamp := pcommon.Timestamp(1684617382541971000)
	endTimestamp := pcommon.Timestamp(1684623646539558000)
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
	pc := sp.Profiles().AppendEmpty()
	pc.SetProfileID(profileID)
	pc.SetStartTime(startTimestamp)
	pc.SetEndTime(endTimestamp)
	pc.Attributes().PutStr("hello", "world")
	pc.Attributes().PutStr("foo", "bar")
	pc.SetDroppedAttributesCount(1)

	pro := pc.Profile()
	// Add sample type
	st := pro.SampleType().AppendEmpty()
	st.SetType(1)
	st.SetUnit(2)
	// Add samples
	s := pro.Sample().AppendEmpty()
	s.LocationIndex().Append(1)
	s.SetLocationsStartIndex(1)
	s.SetLocationsLength(10)
	s.SetStacktraceIdIndex(1)
	s.Value().Append(3)
	s.SetLink(42)
	s.Attributes().Append(1)
	la := s.Label().AppendEmpty()
	la.SetKey(1)
	la.SetStr(2)
	la.SetNum(3)
	la.SetNumUnit(4)
	s.TimestampsUnixNano().Append(12345)
	// Add mappings
	m := pro.Mapping().AppendEmpty()
	m.SetID(1)
	m.SetMemoryStart(2)
	m.SetMemoryLimit(3)
	m.SetFileOffset(4)
	m.SetFilename(5)
	m.SetBuildID(6)
	m.SetBuildIDKind(otlpprofiles.BuildIdKind_BUILD_ID_LINKER)
	m.Attributes().Append(7)
	m.Attributes().Append(8)
	m.SetHasFunctions(true)
	m.SetHasFilenames(true)
	m.SetHasLineNumbers(true)
	m.SetHasInlineFrames(true)
	// Add location
	l := pro.Location().AppendEmpty()
	l.SetID(2)
	l.SetMappingIndex(3)
	l.SetAddress(4)
	l.SetIsFolded(true)
	l.SetTypeIndex(5)
	l.Attributes().Append(6)
	l.Attributes().Append(7)
	li := l.Line().AppendEmpty()
	li.SetFunctionIndex(1)
	li.SetLine(2)
	li.SetColumn(3)
	pro.LocationIndices().Append(1)
	// Add function
	f := pro.Function().AppendEmpty()
	f.SetID(1)
	f.SetName(2)
	f.SetSystemName(3)
	f.SetFilename(4)
	f.SetStartLine(5)
	// Add attribute table
	at := pro.AttributeTable()
	at.PutInt("answer", 42)
	// Add attribute units
	au := pro.AttributeUnits().AppendEmpty()
	au.SetAttributeKey(1)
	au.SetUnit(5)
	// Add link table
	lt := pro.LinkTable().AppendEmpty()
	lt.SetTraceID(traceID)
	lt.SetSpanID(spanID)
	// Add string table
	pro.StringTable().Append("foobar")
	pro.SetDropFrames(1)
	pro.SetKeepFrames(2)
	pro.SetStartTime(1234)
	pro.SetDuration(5678)
	pro.PeriodType().SetType(1)
	pro.PeriodType().SetUnit(2)
	pro.SetPeriod(3)
	pro.Comment().Append(1)
	pro.Comment().Append(2)
	pro.SetDefaultSampleType(4)

	return pd
}()

var profilesJSON = `{"resourceProfiles":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}},{"key":"service.name","value":{"stringValue":"testService"}}],"droppedAttributesCount":1},"scopeProfiles":[{"scope":{"name":"scope name","version":"scope version"},"profiles":[{"profileId":"0102030405060708090a0b0c0d0e0f10","startTimeUnixNano":"1684617382541971000","endTimeUnixNano":"1684623646539558000","attributes":[{"key":"hello","value":{"stringValue":"world"}},{"key":"foo","value":{"stringValue":"bar"}}],"droppedAttributesCount":1,"profile":{"sampleType":[{"type":"1","unit":"2"}],"sample":[{"locationIndex":["1"],"locationsStartIndex":"1","locationsLength":"10","stacktraceIdIndex":1,"value":["3"],"label":[{"key":"1","str":"2","num":"3","numUnit":"4"}],"attributes":["1"],"link":"42","timestampsUnixNano":["12345"]}],"mapping":[{"id":"1","memoryStart":"2","memoryLimit":"3","fileOffset":"4","filename":"5","buildId":"6","attributes":["7","8"],"hasFunctions":true,"hasFilenames":true,"hasLineNumbers":true,"hasInlineFrames":true}],"location":[{"id":"2","mappingIndex":"3","address":"4","line":[{"functionIndex":"1","line":"2","column":"3"}],"isFolded":true,"typeIndex":5,"attributes":["6","7"]}],"locationIndices":["1"],"function":[{"id":"1","name":"2","systemName":"3","filename":"4","startLine":"5"}],"attributeTable":[{"key":"answer","value":{"intValue":"42"}}],"attributeUnits":[{"attributeKey":"1","unit":"5"}],"linkTable":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718"}],"stringTable":["foobar"],"dropFrames":"1","keepFrames":"2","timeNanos":"1234","durationNanos":"5678","periodType":{"type":"1","unit":"2"},"period":"3","comment":["1","2"],"defaultSampleType":"4"}}],"schemaUrl":"schemaURL"}],"schemaUrl":"schemaURL"}]}`

func TestJSONUnmarshal(t *testing.T) {
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalProfiles([]byte(profilesJSON))
	require.NoError(t, err)
	assert.EqualValues(t, profilesOTLP, got)
}

func TestJSONMarshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	require.NoError(t, err)
	assert.Equal(t, profilesJSON, string(jsonBuf))
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
	NewProfileContainer().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse profile_id")
	}
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
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse trace_id")
	}
}

func TestUnmarshalJsoniterSpanLinkInvalidSpanIDField(t *testing.T) {
	jsonStr := `{"span_id":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewLink().unmarshalJsoniter(iter)
	if assert.Error(t, iter.Error) {
		assert.Contains(t, iter.Error.Error(), "parse span_id")
	}
}

func TestUnmarshalJsoniterLabel(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewLabel()
	val.unmarshalJsoniter(iter)
	require.NoError(t, iter.Error)
	assert.Equal(t, NewLabel(), val)
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
