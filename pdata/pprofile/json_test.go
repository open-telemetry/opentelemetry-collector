// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"

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
	// Add sample type
	st := pc.Profile().SampleType().AppendEmpty()
	st.SetType(1)
	st.SetUnit(2)
	// Add samples
	s := pc.Profile().Sample().AppendEmpty()
	s.SetLocationsStartIndex(1)
	s.SetLocationsLength(10)
	s.SetStacktraceIdIndex(1)
	s.SetLink(42)
	s.Attributes().Append(1)
	// Add mappings
	m := pc.Profile().Mapping().AppendEmpty()
	m.SetID(1)
	// Add location
	l := pc.Profile().Location().AppendEmpty()
	l.SetID(2)
	pc.Profile().LocationIndices().Append(1)
	// Add function
	f := pc.Profile().Function().AppendEmpty()
	f.SetName(3)
	// Add attribute table
	at := pc.Profile().AttributeTable()
	at.PutInt("answer", 42)
	// Add attribute units
	au := pc.Profile().AttributeUnits().AppendEmpty()
	au.SetUnit(5)
	// Add link table
	lt := pc.Profile().LinkTable().AppendEmpty()
	lt.SetTraceID(traceID)
	lt.SetSpanID(spanID)
	// Add string table
	pc.Profile().StringTable().Append("foobar")

	return pd
}()

var profilesJSON = `{"resourceProfiles":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}},{"key":"service.name","value":{"stringValue":"testService"}}],"droppedAttributesCount":1},"scopeProfiles":[{"scope":{"name":"scope name","version":"scope version"},"profiles":[{"profileId":"0102030405060708090a0b0c0d0e0f10","startTimeUnixNano":"1684617382541971000","endTimeUnixNano":"1684623646539558000","attributes":[{"key":"hello","value":{"stringValue":"world"}},{"key":"foo","value":{"stringValue":"bar"}}],"droppedAttributesCount":1,"profile":{"sampleType":[{"type":"1","unit":"2"}],"sample":[{"locationsStartIndex":"1","locationsLength":"10","stacktraceIdIndex":1,"attributes":["1"],"link":"42"}],"mapping":[{"id":"1"}],"location":[{"id":"2"}],"locationIndices":["1"],"function":[{"name":"3"}],"attributeTable":[{"key":"answer","value":{"intValue":"42"}}],"attributeUnits":[{"unit":"5"}],"linkTable":[{"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718"}],"stringTable":["foobar"],"periodType":{}}}],"schemaUrl":"schemaURL"}],"schemaUrl":"schemaURL"}]}`

func TestJSONUnmarshal(t *testing.T) {
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalProfiles([]byte(profilesJSON))
	assert.NoError(t, err)
	assert.EqualValues(t, profilesOTLP, got)
}

func TestJSONMarshal(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	assert.NoError(t, err)
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
	assert.NoError(t, iter.Error)
	assert.Equal(t, 1, val.ResourceProfiles().Len())
}

func TestUnmarshalJsoniterResourceProfiles(t *testing.T) {
	jsonStr := `{"extra":"", "resource": {}, "scopeProfiles": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewResourceProfiles()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewResourceProfiles(), val)
}

func TestUnmarshalJsoniterScopeProfiles(t *testing.T) {
	jsonStr := `{"extra":"", "scope": {}, "profiles": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeProfiles()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewScopeProfiles(), val)
}

func TestUnmarshalJsoniterSample(t *testing.T) {
	jsonStr := `{"extra":""}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewSample()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewSample(), val)
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	b.ReportAllocs()

	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	assert.NoError(b, err)
	decoder := &JSONUnmarshaler{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := decoder.UnmarshalProfiles(jsonBuf)
			assert.NoError(b, err)
		}
	})
}
