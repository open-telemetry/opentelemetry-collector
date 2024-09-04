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
	// Add profiles
	pc := sp.Profiles().AppendEmpty()
	pc.SetStartTime(startTimestamp)
	pc.SetEndTime(endTimestamp)
	// Add samples
	s := pc.Profile().Sample().AppendEmpty()
	s.SetLocationsStartIndex(1)
	s.SetLocationsLength(10)
	s.SetStacktraceIdIndex(1)
	s.SetLink(42)
	s.Attributes().Append(1)
	return pd
}()

var profilesJSON = `{"resourceProfiles":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}},{"key":"service.name","value":{"stringValue":"testService"}}],"droppedAttributesCount":1},"scopeProfiles":[{"scope":{},"profiles":[{"startTimeUnixNano":"1684617382541971000","endTimeUnixNano":"1684623646539558000","profile":{"sample":[{"locationsStartIndex":"1","locationsLength":"10","stacktraceIdIndex":1,"attributes":["1"],"link":"42"}],"periodType":{}}}],"schemaUrl":"schemaURL"}],"schemaUrl":"schemaURL"}]}`

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
