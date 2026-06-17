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
	for _, tt := range []struct {
		name        string
		srcProfiles Profiles
		dstProfiles Profiles

		// Expected results after merge
		expectedDictionarySizes struct {
			StringTable    int
			AttributeTable int
			StackTable     int
			LocationTable  int
			FunctionTable  int
			MappingTable   int
			LinkTable      int
		}
		expectedProfileCount int
	}{
		{
			name: "Empty Profiles",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    1, // Just the empty string
				AttributeTable: 1,
				StackTable:     1,
				LocationTable:  1,
				FunctionTable:  1,
				MappingTable:   1,
				LinkTable:      1,
			},
			expectedProfileCount: 0,
			srcProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
			dstProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
		},
		{
			name: "Single Profile",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    7, // empty + 6 strings
				AttributeTable: 2, // empty + a1
				StackTable:     2, // empty + st1
				LocationTable:  3, // empty + loc1 + loc2
				FunctionTable:  1,
				MappingTable:   1,
				LinkTable:      1,
			},
			expectedProfileCount: 1,
			srcProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("sample-type")     // 1
				ps.Dictionary().StringTable().Append("sample-unit")     // 2
				ps.Dictionary().StringTable().Append("period-type")     // 3
				ps.Dictionary().StringTable().Append("period-unit")     // 4
				ps.Dictionary().StringTable().Append("attribute1-key")  // 5
				ps.Dictionary().StringTable().Append("attribute1-unit") // 6

				a1 := ps.Dictionary().AttributeTable().AppendEmpty()
				a1.SetKeyStrindex(5)
				a1.SetUnitStrindex(6)
				a1.Value().SetStr("AnyValue")

				st1 := ps.Dictionary().StackTable().AppendEmpty()
				st1.LocationIndices().Append(1, 2)

				loc1 := ps.Dictionary().LocationTable().AppendEmpty()
				loc1.SetAddress(1337)
				loc2 := ps.Dictionary().LocationTable().AppendEmpty()
				ln1 := loc2.Lines().AppendEmpty()
				ln1.SetLine(42)

				rp := ps.ResourceProfiles().AppendEmpty()
				rp.SetSchemaUrl("resource-schema-url")
				rp.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp := rp.ScopeProfiles().AppendEmpty()
				sp.SetSchemaUrl("scope-schema-url")

				p := sp.Profiles().AppendEmpty()
				p.SampleType().SetTypeStrindex(1)
				p.SampleType().SetUnitStrindex(2)
				p.PeriodType().SetTypeStrindex(3)
				p.PeriodType().SetUnitStrindex(4)

				s1 := p.Samples().AppendEmpty()
				s1.AttributeIndices().Append(1)
				s1.SetStackIndex(1)

				return ps
			}(),
			dstProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
		},
		{
			name: "Multiple Profile",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    7, // empty + 6 strings
				AttributeTable: 1,
				StackTable:     1,
				LocationTable:  1,
				FunctionTable:  1,
				MappingTable:   1,
				LinkTable:      1,
			},
			expectedProfileCount: 3,
			srcProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("sample-type-1") // 1
				ps.Dictionary().StringTable().Append("sample-unit-1") // 2
				ps.Dictionary().StringTable().Append("sample-type-2") // 3
				ps.Dictionary().StringTable().Append("sample-unit-2") // 4
				ps.Dictionary().StringTable().Append("sample-type-3") // 5
				ps.Dictionary().StringTable().Append("sample-unit-3") // 6

				rp := ps.ResourceProfiles().AppendEmpty()
				rp.SetSchemaUrl("resource-schema-url")
				rp.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp := rp.ScopeProfiles().AppendEmpty()
				sp.SetSchemaUrl("scope-schema-url")

				p1 := sp.Profiles().AppendEmpty()
				p1.SampleType().SetTypeStrindex(1)
				p1.SampleType().SetUnitStrindex(2)

				p2 := sp.Profiles().AppendEmpty()
				p2.SampleType().SetTypeStrindex(3)
				p2.SampleType().SetUnitStrindex(4)

				p3 := sp.Profiles().AppendEmpty()
				p3.SampleType().SetTypeStrindex(5)
				p3.SampleType().SetUnitStrindex(6)

				return ps
			}(),
			dstProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
		},
		{
			name: "Multiple Profile with partly prepopulated destination",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    10, // empty + 3 unrelated + 6 from src
				AttributeTable: 1,
				StackTable:     1,
				LocationTable:  1,
				FunctionTable:  1,
				MappingTable:   1,
				LinkTable:      1,
			},
			expectedProfileCount: 3,
			srcProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("sample-type-1") // 1
				ps.Dictionary().StringTable().Append("sample-unit-1") // 2
				ps.Dictionary().StringTable().Append("sample-type-2") // 3
				ps.Dictionary().StringTable().Append("sample-unit-2") // 4
				ps.Dictionary().StringTable().Append("sample-type-3") // 5
				ps.Dictionary().StringTable().Append("sample-unit-3") // 6

				rp := ps.ResourceProfiles().AppendEmpty()
				rp.SetSchemaUrl("resource-schema-url")
				rp.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp := rp.ScopeProfiles().AppendEmpty()
				sp.SetSchemaUrl("scope-schema-url")

				p1 := sp.Profiles().AppendEmpty()
				p1.SampleType().SetTypeStrindex(1)
				p1.SampleType().SetUnitStrindex(2)

				p2 := sp.Profiles().AppendEmpty()
				p2.SampleType().SetTypeStrindex(3)
				p2.SampleType().SetUnitStrindex(4)

				p3 := sp.Profiles().AppendEmpty()
				p3.SampleType().SetTypeStrindex(5)
				p3.SampleType().SetUnitStrindex(6)

				return ps
			}(),
			dstProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("unrelated-1") // 1
				ps.Dictionary().StringTable().Append("unrelated-2") // 2
				ps.Dictionary().StringTable().Append("unrelated-3") // 3
				return ps
			}(),
		},
		{
			name: "Multiple Profile with reused samples",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    9, // empty + 8 unique strings after merge
				AttributeTable: 1,
				StackTable:     3, // empty + 2 unique stacks
				LocationTable:  3, // empty + 2 unique locations
				FunctionTable:  2, // empty + fn1
				MappingTable:   1,
				LinkTable:      1,
			},
			expectedProfileCount: 3,
			srcProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("sample-type-1")  // 1
				ps.Dictionary().StringTable().Append("sample-unit-1")  // 2
				ps.Dictionary().StringTable().Append("sample-type-2")  // 3
				ps.Dictionary().StringTable().Append("sample-unit-2")  // 4
				ps.Dictionary().StringTable().Append("sample-type-3")  // 5
				ps.Dictionary().StringTable().Append("sample-unit-3")  // 6
				ps.Dictionary().StringTable().Append("filename-1")     // 7
				ps.Dictionary().StringTable().Append("functionname-1") // 8

				st1 := ps.Dictionary().StackTable().AppendEmpty()
				st1.LocationIndices().Append(1)

				st2 := ps.Dictionary().StackTable().AppendEmpty()
				st2.LocationIndices().Append(2)

				loc1 := ps.Dictionary().LocationTable().AppendEmpty()
				loc1.SetAddress(42)

				loc2 := ps.Dictionary().LocationTable().AppendEmpty()
				ln1 := loc2.Lines().AppendEmpty()
				ln1.SetFunctionIndex(1)
				ln1.SetLine(1337)

				fn1 := ps.Dictionary().FunctionTable().AppendEmpty()
				fn1.SetFilenameStrindex(7)
				fn1.SetNameStrindex(8)

				rp := ps.ResourceProfiles().AppendEmpty()
				rp.SetSchemaUrl("resource-schema-url")
				rp.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp := rp.ScopeProfiles().AppendEmpty()
				sp.SetSchemaUrl("scope-schema-url")

				p1 := sp.Profiles().AppendEmpty()
				p1.SampleType().SetTypeStrindex(1)
				p1.SampleType().SetUnitStrindex(2)
				s1 := p1.Samples().AppendEmpty()
				s1.SetStackIndex(1)

				p2 := sp.Profiles().AppendEmpty()
				p2.SampleType().SetTypeStrindex(3)
				p2.SampleType().SetUnitStrindex(4)
				s2 := p2.Samples().AppendEmpty()
				s2.SetStackIndex(2)

				p3 := sp.Profiles().AppendEmpty()
				p3.SampleType().SetTypeStrindex(5)
				p3.SampleType().SetUnitStrindex(6)
				s3 := p3.Samples().AppendEmpty()
				s3.SetStackIndex(1)

				return ps
			}(),
			dstProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
		},
		{
			name: "Multiple ResourceProfiles with reused samples",
			expectedDictionarySizes: struct {
				StringTable    int
				AttributeTable int
				StackTable     int
				LocationTable  int
				FunctionTable  int
				MappingTable   int
				LinkTable      int
			}{
				StringTable:    9, // empty + 8 unique strings
				AttributeTable: 1,
				StackTable:     3, // empty + 2 unique stacks
				LocationTable:  3, // empty + 2 unique locations
				FunctionTable:  2, // empty + fn1
				MappingTable:   2, // empty + m1
				LinkTable:      1,
			},
			expectedProfileCount: 6,
			srcProfiles: func() Profiles {
				ps := NewProfiles()

				// Make sure we are conform with the protocol
				ps.Dictionary().MappingTable().AppendEmpty()
				ps.Dictionary().LocationTable().AppendEmpty()
				ps.Dictionary().FunctionTable().AppendEmpty()
				ps.Dictionary().LinkTable().AppendEmpty()
				ps.Dictionary().StringTable().Append("")
				ps.Dictionary().AttributeTable().AppendEmpty()
				ps.Dictionary().StackTable().AppendEmpty()

				ps.Dictionary().StringTable().Append("sample-type-1")  // 1
				ps.Dictionary().StringTable().Append("sample-unit-1")  // 2
				ps.Dictionary().StringTable().Append("sample-type-2")  // 3
				ps.Dictionary().StringTable().Append("sample-unit-2")  // 4
				ps.Dictionary().StringTable().Append("sample-type-3")  // 5
				ps.Dictionary().StringTable().Append("sample-unit-3")  // 6
				ps.Dictionary().StringTable().Append("filename-1")     // 7
				ps.Dictionary().StringTable().Append("functionname-1") // 8

				st1 := ps.Dictionary().StackTable().AppendEmpty()
				st1.LocationIndices().Append(1)

				st2 := ps.Dictionary().StackTable().AppendEmpty()
				st2.LocationIndices().Append(2)

				loc1 := ps.Dictionary().LocationTable().AppendEmpty()
				loc1.SetAddress(42)
				loc1.SetMappingIndex(1)

				loc2 := ps.Dictionary().LocationTable().AppendEmpty()
				ln1 := loc2.Lines().AppendEmpty()
				ln1.SetFunctionIndex(1)
				ln1.SetLine(1337)

				fn1 := ps.Dictionary().FunctionTable().AppendEmpty()
				fn1.SetFilenameStrindex(7)
				fn1.SetNameStrindex(8)

				m1 := ps.Dictionary().MappingTable().AppendEmpty()
				m1.SetFilenameStrindex(8)

				rp1 := ps.ResourceProfiles().AppendEmpty()
				rp1.SetSchemaUrl("resource-schema-url")
				rp1.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp1 := rp1.ScopeProfiles().AppendEmpty()
				sp1.SetSchemaUrl("scope-schema-url")

				p11 := sp1.Profiles().AppendEmpty()
				p11.SampleType().SetTypeStrindex(1)
				p11.SampleType().SetUnitStrindex(2)
				s11 := p11.Samples().AppendEmpty()
				s11.SetStackIndex(1)

				p12 := sp1.Profiles().AppendEmpty()
				p12.SampleType().SetTypeStrindex(3)
				p12.SampleType().SetUnitStrindex(4)
				s12 := p12.Samples().AppendEmpty()
				s12.SetStackIndex(2)

				p13 := sp1.Profiles().AppendEmpty()
				p13.SampleType().SetTypeStrindex(5)
				p13.SampleType().SetUnitStrindex(6)
				s13 := p13.Samples().AppendEmpty()
				s13.SetStackIndex(1)

				rp2 := ps.ResourceProfiles().AppendEmpty()
				rp2.SetSchemaUrl("resource-schema-url")
				rp2.Resource().Attributes().PutStr("resource-attribute-key",
					"resource-attribute-value")

				sp2 := rp2.ScopeProfiles().AppendEmpty()
				sp2.SetSchemaUrl("scope-schema-url")

				p21 := sp2.Profiles().AppendEmpty()
				p21.SampleType().SetTypeStrindex(1)
				p21.SampleType().SetUnitStrindex(2)
				s21 := p21.Samples().AppendEmpty()
				s21.SetStackIndex(1)

				p22 := sp2.Profiles().AppendEmpty()
				p22.SampleType().SetTypeStrindex(3)
				p22.SampleType().SetUnitStrindex(4)
				s22 := p22.Samples().AppendEmpty()
				s22.SetStackIndex(2)

				p23 := sp2.Profiles().AppendEmpty()
				p23.SampleType().SetTypeStrindex(5)
				p23.SampleType().SetUnitStrindex(6)
				s23 := p23.Samples().AppendEmpty()
				s23.SetStackIndex(1)

				return ps
			}(),
			dstProfiles: func() Profiles {
				p := NewProfiles()

				// Make sure we are conform with the protocol
				p.Dictionary().MappingTable().AppendEmpty()
				p.Dictionary().LocationTable().AppendEmpty()
				p.Dictionary().FunctionTable().AppendEmpty()
				p.Dictionary().LinkTable().AppendEmpty()
				p.Dictionary().StringTable().Append("")
				p.Dictionary().AttributeTable().AppendEmpty()
				p.Dictionary().StackTable().AppendEmpty()
				return p
			}(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srcProfiles := tt.srcProfiles
			dstProfiles := tt.dstProfiles
			err := srcProfiles.MergeTo(dstProfiles)
			require.NoError(t, err)

			// Verify dictionary sizes
			assert.Equal(t, tt.expectedDictionarySizes.StringTable, dstProfiles.Dictionary().StringTable().Len(),
				"StringTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.AttributeTable, dstProfiles.Dictionary().AttributeTable().Len(),
				"AttributeTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.StackTable, dstProfiles.Dictionary().StackTable().Len(),
				"StackTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.LocationTable, dstProfiles.Dictionary().LocationTable().Len(),
				"LocationTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.FunctionTable, dstProfiles.Dictionary().FunctionTable().Len(),
				"FunctionTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.MappingTable, dstProfiles.Dictionary().MappingTable().Len(),
				"MappingTable size mismatch")
			assert.Equal(t, tt.expectedDictionarySizes.LinkTable, dstProfiles.Dictionary().LinkTable().Len(),
				"LinkTable size mismatch")

			// Verify profile count
			totalProfiles := 0
			for _, rp := range dstProfiles.ResourceProfiles().All() {
				for _, sp := range rp.ScopeProfiles().All() {
					totalProfiles += sp.Profiles().Len()
				}
			}
			assert.Equal(t, tt.expectedProfileCount, totalProfiles, "Total profile count mismatch")
		})
	}
}

func TestProfilesMergeToSelf(t *testing.T) {
	profiles := NewProfiles()
	profiles.Dictionary().StringTable().Append("", "test")
	profiles.ResourceProfiles().AppendEmpty()

	require.NoError(t, profiles.MergeTo(profiles))

	assert.Equal(t, 2, profiles.Dictionary().StringTable().Len())
	assert.Equal(t, 1, profiles.ResourceProfiles().Len())
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
		Samples().AppendEmpty()
	sample.SetStackIndex(1)

	err := src.MergeTo(dest)
	require.Error(t, err)

	assert.Equal(t, 0, dest.ResourceProfiles().Len())
}

// TestProfilesMergeTo_ResourceAttributeRoundTrip verifies that resource and
// scope attributes survive a marshal→unmarshal→merge→marshal→unmarshal
// round-trip without corruption of KeyStrindex references.
// This is the core scenario from
// https://github.com/open-telemetry/opentelemetry-collector/issues/15084.
func TestProfilesMergeTo_ResourceAttributeRoundTrip(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}

	// Build two spec-conformant profiles with DIFFERENT string tables so that
	// after merge the destination string table differs from the source's
	// original indices, which is what triggers the stale-KeyStrindex bug.
	buildProfile := func(periodType, periodUnit, serviceName, extraKey, extraVal string) Profiles {
		p := NewProfiles()
		// Sentinel entries at index 0 (required by spec).
		p.Dictionary().StringTable().Append("")
		p.Dictionary().AttributeTable().AppendEmpty()
		p.Dictionary().StackTable().AppendEmpty()
		p.Dictionary().LocationTable().AppendEmpty()
		p.Dictionary().FunctionTable().AppendEmpty()
		p.Dictionary().MappingTable().AppendEmpty()
		p.Dictionary().LinkTable().AppendEmpty()

		p.Dictionary().StringTable().Append(periodType) // 1
		p.Dictionary().StringTable().Append(periodUnit) // 2

		rp := p.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutStr("service.name", serviceName)
		rp.Resource().Attributes().PutStr(extraKey, extraVal)

		sp := rp.ScopeProfiles().AppendEmpty()
		sp.Scope().Attributes().PutStr("scope.version", "v1")

		pr := sp.Profiles().AppendEmpty()
		pr.PeriodType().SetTypeStrindex(1)
		pr.PeriodType().SetUnitStrindex(2)
		return p
	}

	profileA := buildProfile("cpu", "nanoseconds", "service-A", "deployment", "prod")
	profileB := buildProfile("memory", "bytes", "service-B", "host.name", "host-1")

	// Record original attributes before any serialization.
	origResA := attrMapToStrings(profileA.ResourceProfiles().At(0).Resource().Attributes())
	origResB := attrMapToStrings(profileB.ResourceProfiles().At(0).Resource().Attributes())
	origScopeA := attrMapToStrings(profileA.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().Attributes())
	origScopeB := attrMapToStrings(profileB.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().Attributes())

	// Marshal → Unmarshal (simulates Kafka write/read).
	bytesA, err := marshaler.MarshalProfiles(profileA)
	require.NoError(t, err)
	bytesB, err := marshaler.MarshalProfiles(profileB)
	require.NoError(t, err)

	dstProfiles, err := unmarshaler.UnmarshalProfiles(bytesA)
	require.NoError(t, err)
	srcProfiles, err := unmarshaler.UnmarshalProfiles(bytesB)
	require.NoError(t, err)

	// MergeTo — merges src into dst, which changes the dictionary.
	require.NoError(t, srcProfiles.MergeTo(dstProfiles))

	// Re-marshal the merged result.
	mergedBytes, err := marshaler.MarshalProfiles(dstProfiles)
	require.NoError(t, err)

	// Re-unmarshal (simulates consumer reading from Kafka).
	finalProfiles, err := unmarshaler.UnmarshalProfiles(mergedBytes)
	require.NoError(t, err)

	// After the full round-trip, resource and scope attributes must match.
	require.Equal(t, 2, finalProfiles.ResourceProfiles().Len(),
		"merged profiles should have 2 resource profiles")

	finalResA := attrMapToStrings(finalProfiles.ResourceProfiles().At(0).Resource().Attributes())
	finalResB := attrMapToStrings(finalProfiles.ResourceProfiles().At(1).Resource().Attributes())
	assert.Equal(t, origResA, finalResA,
		"first resource attributes corrupted after round-trip")
	assert.Equal(t, origResB, finalResB,
		"second resource attributes corrupted after round-trip")

	finalScopeA := attrMapToStrings(finalProfiles.ResourceProfiles().At(0).ScopeProfiles().At(0).Scope().Attributes())
	finalScopeB := attrMapToStrings(finalProfiles.ResourceProfiles().At(1).ScopeProfiles().At(0).Scope().Attributes())
	assert.Equal(t, origScopeA, finalScopeA,
		"first scope attributes corrupted after round-trip")
	assert.Equal(t, origScopeB, finalScopeB,
		"second scope attributes corrupted after round-trip")
}

func attrMapToStrings(m pcommon.Map) map[string]string {
	result := make(map[string]string, m.Len())
	m.Range(func(k string, v pcommon.Value) bool {
		result[k] = v.AsString()
		return true
	})
	return result
}
