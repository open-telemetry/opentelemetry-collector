// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	gootlpprofiles "go.opentelemetry.io/proto/slim/otlp/profiles/v1development"
	goproto "google.golang.org/protobuf/proto"
)

func TestProfilesProtoWireCompatibility(t *testing.T) {
	// This test verifies that OTLP ProtoBufs generated using goproto lib in
	// opentelemetry-proto repository OTLP ProtoBufs generated using gogoproto lib in
	// this repository are wire compatible.

	// Generate Profiles as pdata struct.
	td := generateTestProfiles()

	// Marshal its underlying ProtoBuf to wire.
	marshaler := &ProtoMarshaler{}
	wire1, err := marshaler.MarshalProfiles(td)
	require.NoError(t, err)
	assert.NotNil(t, wire1)

	// Unmarshal from the wire to OTLP Protobuf in goproto's representation.
	var goprotoMessage gootlpprofiles.ProfilesData
	err = goproto.Unmarshal(wire1, &goprotoMessage)
	require.NoError(t, err)

	// Marshal to the wire again.
	wire2, err := goproto.Marshal(&goprotoMessage)
	require.NoError(t, err)
	assert.NotNil(t, wire2)

	// Unmarshal from the wire into gogoproto's representation.
	var td2 Profiles
	unmarshaler := &ProtoUnmarshaler{}
	td2, err = unmarshaler.UnmarshalProfiles(wire2)
	require.NoError(t, err)

	// After unmarshal, td2 will have resolved references (strings instead of string_value_ref/key_ref)
	// while td may have references. Marshal td2 again to verify wire compatibility.
	wire3, err := marshaler.MarshalProfiles(td2)
	require.NoError(t, err)

	// Verify that wire1 and wire3 are compatible by unmarshaling both and checking the data
	var check1, check2 gootlpprofiles.ProfilesData
	require.NoError(t, goproto.Unmarshal(wire1, &check1))
	require.NoError(t, goproto.Unmarshal(wire3, &check2))

	// Both should unmarshal successfully, proving wire compatibility
	assert.NotNil(t, check1.ResourceProfiles)
	assert.NotNil(t, check2.ResourceProfiles)
	assert.Len(t, check1.ResourceProfiles, len(check2.ResourceProfiles))
}

func TestProtoProfilesUnmarshalerError(t *testing.T) {
	p := &ProtoUnmarshaler{}
	_, err := p.UnmarshalProfiles([]byte("+$%"))
	assert.Error(t, err)
}

func TestProtoSizer(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	td := NewProfiles()
	td.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	td.Dictionary().StringTable().Append("foobar")

	size := marshaler.ProfilesSize(td)

	bytes, err := marshaler.MarshalProfiles(td)
	require.NoError(t, err)
	assert.Equal(t, len(bytes), size)
}

func TestProtoSizerEmptyProfiles(t *testing.T) {
	sizer := &ProtoMarshaler{}
	assert.Equal(t, 2, sizer.ProfilesSize(NewProfiles()))
}

func BenchmarkProfilesToProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	profiles := generateBenchmarkProfiles(128)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		buf, err := marshaler.MarshalProfiles(profiles)
		require.NoError(b, err)
		assert.NotEmpty(b, buf)
	}
}

func BenchmarkProfilesFromProto(b *testing.B) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}
	baseProfiles := generateBenchmarkProfiles(128)
	buf, err := marshaler.MarshalProfiles(baseProfiles)
	require.NoError(b, err)
	assert.NotEmpty(b, buf)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		profiles, err := unmarshaler.UnmarshalProfiles(buf)
		require.NoError(b, err)
		assert.Equal(b, baseProfiles.ResourceProfiles().Len(), profiles.ResourceProfiles().Len())
	}
}

func generateBenchmarkProfiles(samplesCount int) Profiles {
	md := NewProfiles()
	ilm := md.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	ilm.Samples().EnsureCapacity(samplesCount)
	for range samplesCount {
		im := ilm.Samples().AppendEmpty()
		im.SetStackIndex(0)
	}
	return md
}

// generateProfiles creates a Profiles object with the specified number of resources, scopes, profiles, and samples.
func generateProfiles(b *testing.B, resourceCount, scopeCount, profileCount, sampleCount int) Profiles {
	b.Helper()

	profiles := NewProfiles()
	dict := profiles.Dictionary()

	// Pre-populate dictionary with common strings
	dict.StringTable().Append("") // Index 0 is always empty string
	dict.StringTable().Append("cpu")
	dict.StringTable().Append("nanoseconds")
	dict.StringTable().Append("samples")
	dict.StringTable().Append("count")

	// Generate resource profiles
	for r := range resourceCount {
		rp := profiles.ResourceProfiles().AppendEmpty()
		rp.SetSchemaUrl(semconv.SchemaURL)
		resource := rp.Resource()

		// Add resource attributes
		attrs := resource.Attributes()
		attrs.PutStr(string(semconv.ServiceNameKey), fmt.Sprintf("service-%d", r))
		attrs.PutStr(string(semconv.ServiceVersionKey), fmt.Sprintf("version-%d", r))
		attrs.PutStr(string(semconv.ProcessPIDKey), strconv.Itoa(1000+r))
		attrs.PutStr(string(semconv.K8SPodNameKey), fmt.Sprintf("pod-%d", r%10))
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), "default")
		attrs.PutStr(string(semconv.TelemetrySDKNameKey), "opentelemetry")

		// Generate scope profiles
		for s := range scopeCount {
			sp := rp.ScopeProfiles().AppendEmpty()
			sp.SetSchemaUrl(semconv.SchemaURL)
			scope := sp.Scope()
			scope.SetName(fmt.Sprintf("profiler-scope-%d", s))
			scope.SetVersion("1.0.0")

			// Generate profiles
			for range profileCount {
				profile := sp.Profiles().AppendEmpty()

				// Add sample types
				sampleType := profile.SampleType()
				sampleType.SetTypeStrindex(1) // "cpu"
				sampleType.SetUnitStrindex(2) // "nanoseconds"

				// Add period type
				periodType := profile.PeriodType()
				periodType.SetTypeStrindex(1) // "cpu"
				periodType.SetUnitStrindex(2) // "nanoseconds"
				profile.SetPeriod(1000000)

				// Generate samples
				samples := profile.Samples()
				for i := range sampleCount {
					sample := samples.AppendEmpty()
					sample.SetStackIndex(int32(i % 100))

					// Add attribute indices for samples
					sample.AttributeIndices().Append(int32(i % 10))
				}
			}
		}
	}

	return profiles
}

func BenchmarkUnmarshalProfiles(b *testing.B) {
	testCases := []struct {
		name          string
		resourceCount int
		scopeCount    int
		profileCount  int
		sampleCount   int
	}{
		{
			name:          "small",
			resourceCount: 1,
			scopeCount:    1,
			profileCount:  1,
			sampleCount:   100,
		},
		{
			name:          "medium",
			resourceCount: 5,
			scopeCount:    2,
			profileCount:  2,
			sampleCount:   500,
		},
		{
			name:          "large",
			resourceCount: 20,
			scopeCount:    3,
			profileCount:  5,
			sampleCount:   1000,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate profile data and marshal it
			profiles := generateProfiles(b, tc.resourceCount, tc.scopeCount, tc.profileCount, tc.sampleCount)
			marshaler := &ProtoMarshaler{}
			data, err := marshaler.MarshalProfiles(profiles)
			if err != nil {
				b.Fatalf("failed to marshal profiles: %v", err)
			}

			unmarshaler := &ProtoUnmarshaler{}
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				profiles, err := unmarshaler.UnmarshalProfiles(data)
				if err != nil {
					b.Fatalf("failed to unmarshal: %v", err)
				}
				_ = profiles
			}
		})
	}
}

func BenchmarkMarshalProfiles(b *testing.B) {
	testCases := []struct {
		name          string
		resourceCount int
		scopeCount    int
		profileCount  int
		sampleCount   int
	}{
		{
			name:          "small",
			resourceCount: 1,
			scopeCount:    1,
			profileCount:  1,
			sampleCount:   100,
		},
		{
			name:          "medium",
			resourceCount: 5,
			scopeCount:    2,
			profileCount:  2,
			sampleCount:   500,
		},
		{
			name:          "large",
			resourceCount: 20,
			scopeCount:    3,
			profileCount:  5,
			sampleCount:   1000,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			marshaler := &ProtoMarshaler{}

			// with_refs: simulate the normal ingest path where data was
			// received on the wire (refs present), then unmarshaled (refs
			// resolved but KeyRef kept), and is now being re-marshaled
			// without any attribute modifications.
			b.Run("with_refs", func(b *testing.B) {
				profiles := generateProfiles(b, tc.resourceCount, tc.scopeCount, tc.profileCount, tc.sampleCount)
				unmarshaler := &ProtoUnmarshaler{}
				buf, err := marshaler.MarshalProfiles(profiles)
				if err != nil {
					b.Fatalf("failed to marshal: %v", err)
				}
				profiles, err = unmarshaler.UnmarshalProfiles(buf)
				if err != nil {
					b.Fatalf("failed to unmarshal: %v", err)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					buf, err := marshaler.MarshalProfiles(profiles)
					if err != nil {
						b.Fatalf("failed to marshal: %v", err)
					}
					_ = buf
				}
			})

			// without_refs: each iteration gets a fresh copy with no refs,
			// simulating data that was constructed or had attributes modified.
			b.Run("without_refs", func(b *testing.B) {
				copies := make([]Profiles, b.N)
				for i := range copies {
					copies[i] = generateProfiles(b, tc.resourceCount, tc.scopeCount, tc.profileCount, tc.sampleCount)
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					buf, err := marshaler.MarshalProfiles(copies[i])
					if err != nil {
						b.Fatalf("failed to marshal: %v", err)
					}
					_ = buf
				}
			})
		})
	}
}
