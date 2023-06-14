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

var _ Marshaler = (*JSONMarshaler)(nil)
var _ Unmarshaler = (*JSONUnmarshaler)(nil)

var profilesOTLP = func() Profiles {
	ld := NewProfiles()
	rl := ld.ResourceProfiles().AppendEmpty()
	rl.Resource().Attributes().PutStr("host.name", "testHost")
	rl.Resource().SetDroppedAttributesCount(1)
	rl.SetSchemaUrl("resource_schema")
	il := rl.ScopeProfiles().AppendEmpty()
	il.Scope().SetName("name")
	il.Scope().SetVersion("version")
	il.Scope().SetDroppedAttributesCount(1)
	il.SetSchemaUrl("scope_schema")
	lg := il.Profiles().AppendEmpty()
	lg.SetDroppedAttributesCount(1)
	profileID := pcommon.ProfileID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	lg.SetProfileID(profileID)
	lg.SetStartTime(pcommon.Timestamp(1684617382541971000))
	lg.SetEndTime(pcommon.Timestamp(1684623646539558000))
	lg.Attributes().PutStr("sdkVersion", "1.0.1")
	return ld
}()

func TestProfilesJSON(t *testing.T) {
	encoder := &JSONMarshaler{}
	jsonBuf, err := encoder.MarshalProfiles(profilesOTLP)
	assert.NoError(t, err)
	decoder := &JSONUnmarshaler{}
	got, err := decoder.UnmarshalProfiles(jsonBuf)
	assert.NoError(t, err)
	assert.EqualValues(t, profilesOTLP, got)
}

var profilesJSON = `{"resourceProfiles":[{"resource":{"attributes":[{"key":"host.name","value":{"stringValue":"testHost"}}],"droppedAttributesCount":1},"scopeProfiles":[{"scope":{"name":"name","version":"version","droppedAttributesCount":1},"profileRecords":[{"timeUnixNano":"1684617382541971000","observedTimeUnixNano":"1684623646539558000","severityNumber":17,"severityText":"Error","body":{"stringValue":"hello world"},"attributes":[{"key":"sdkVersion","value":{"stringValue":"1.0.1"}}],"droppedAttributesCount":1,"flags":1,"traceId":"0102030405060708090a0b0c0d0e0f10","spanId":"1112131415161718"}],"schemaUrl":"scope_schema"}],"schemaUrl":"resource_schema"}]}`

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

func TestUnmarshalJsoniterProfilesData(t *testing.T) {
	jsonStr := `{"extra":"", "resourceProfiles": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewProfiles()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewProfiles(), val)
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
	jsonStr := `{"extra":"", "scope": {}, "profileRecords": []}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewScopeProfiles()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewScopeProfiles(), val)
}

func TestUnmarshalJsoniterProfileRecord(t *testing.T) {
	jsonStr := `{"extra":"", "body":{}}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	val := NewProfile()
	val.unmarshalJsoniter(iter)
	assert.NoError(t, iter.Error)
	assert.Equal(t, NewProfile(), val)
}

func TestUnmarshalJsoniterProfileWrongTraceID(t *testing.T) {
	jsonStr := `{"body":{}, "traceId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewProfile().unmarshalJsoniter(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse trace_id")
}

func TestUnmarshalJsoniterProfileWrongSpanID(t *testing.T) {
	jsonStr := `{"body":{}, "spanId":"--"}`
	iter := jsoniter.ConfigFastest.BorrowIterator([]byte(jsonStr))
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	NewProfile().unmarshalJsoniter(iter)
	require.Error(t, iter.Error)
	assert.Contains(t, iter.Error.Error(), "parse span_id")
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
