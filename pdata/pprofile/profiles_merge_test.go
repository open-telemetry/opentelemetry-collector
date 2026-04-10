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

// TestProfilesMergeTo_StackIndexZero verifies that samples referencing
// StackIndex=0 are correctly remapped during MergeTo, rather than being
// silently skipped (which causes data corruption).
func TestProfilesMergeTo_StackIndexZero(t *testing.T) {
	// Source: sample with StackIndex=0 pointing to a real stack.
	src := NewProfiles()
	src.Dictionary().StringTable().Append("")            // 0
	src.Dictionary().StringTable().Append("srcFuncName") // 1
	src.Dictionary().StringTable().Append("srcFileName") // 2
	src.Dictionary().AttributeTable().AppendEmpty()      // 0: sentinel
	src.Dictionary().LinkTable().AppendEmpty()           // 0: sentinel
	src.Dictionary().MappingTable().AppendEmpty()        // 0: sentinel

	srcFn := src.Dictionary().FunctionTable().AppendEmpty() // 0: real function
	srcFn.SetNameStrindex(1)
	srcFn.SetFilenameStrindex(2)

	srcLoc := src.Dictionary().LocationTable().AppendEmpty() // 0: real location
	srcLoc.SetAddress(42)
	srcLine := srcLoc.Lines().AppendEmpty()
	srcLine.SetFunctionIndex(0) // refs function at index 0

	srcStack := src.Dictionary().StackTable().AppendEmpty() // 0: real stack
	srcStack.LocationIndices().Append(0)                    // refs location at index 0

	rp := src.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr("service.name", "src-service")
	sp := rp.ScopeProfiles().AppendEmpty()
	p := sp.Profiles().AppendEmpty()
	sample := p.Samples().AppendEmpty()
	sample.SetStackIndex(0) // Points to real stack at index 0

	// Destination: has different data at index 0.
	dst := NewProfiles()
	dst.Dictionary().StringTable().Append("")            // 0
	dst.Dictionary().StringTable().Append("dstFuncName") // 1
	dst.Dictionary().StringTable().Append("dstFileName") // 2
	dst.Dictionary().AttributeTable().AppendEmpty()      // 0: sentinel
	dst.Dictionary().LinkTable().AppendEmpty()           // 0: sentinel
	dst.Dictionary().MappingTable().AppendEmpty()        // 0: sentinel

	dstFn := dst.Dictionary().FunctionTable().AppendEmpty() // 0: different function
	dstFn.SetNameStrindex(1)
	dstFn.SetFilenameStrindex(2)

	dstLoc := dst.Dictionary().LocationTable().AppendEmpty() // 0: different location
	dstLoc.SetAddress(99)
	dstLine := dstLoc.Lines().AppendEmpty()
	dstLine.SetFunctionIndex(0)

	dstStack := dst.Dictionary().StackTable().AppendEmpty() // 0: different stack
	dstStack.LocationIndices().Append(0)

	err := src.MergeTo(dst)
	require.NoError(t, err)

	// After merge, the source's sample should reference a stack whose
	// location has Address=42 (from src), NOT Address=99 (from dst).
	require.Equal(t, 1, dst.ResourceProfiles().Len())
	mergedSample := dst.ResourceProfiles().At(0).
		ScopeProfiles().At(0).
		Profiles().At(0).
		Samples().At(0)

	stackIdx := mergedSample.StackIndex()
	stack := dst.Dictionary().StackTable().At(int(stackIdx))
	locIdx := stack.LocationIndices().At(0)
	loc := dst.Dictionary().LocationTable().At(int(locIdx))

	assert.Equal(t, uint64(42), loc.Address(),
		"source sample's stack should reference the original location with address 42, not destination's 99")

	// Also verify the function name resolves to srcFuncName, not dstFuncName.
	line := loc.Lines().At(0)
	fn := dst.Dictionary().FunctionTable().At(int(line.FunctionIndex()))
	fnName := dst.Dictionary().StringTable().At(int(fn.NameStrindex()))
	assert.Equal(t, "srcFuncName", fnName,
		"source sample's function should be srcFuncName, not dstFuncName")
}

// TestProfilesMergeTo_ResourceAttributeRoundTrip verifies that resource
// attributes survive a marshal→unmarshal→merge→marshal→unmarshal round-trip
// without corruption of KeyStrindex references.
func TestProfilesMergeTo_ResourceAttributeRoundTrip(t *testing.T) {
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}

	// Create two profiles with DIFFERENT string tables so that after merge
	// the destination string table differs from the source's original indices.
	profileA := NewProfiles()
	profileA.Dictionary().StringTable().Append("")
	profileA.Dictionary().StringTable().Append("cpu")
	profileA.Dictionary().StringTable().Append("nanoseconds")
	profileA.Dictionary().AttributeTable().AppendEmpty()
	profileA.Dictionary().StackTable().AppendEmpty()
	profileA.Dictionary().LocationTable().AppendEmpty()
	profileA.Dictionary().FunctionTable().AppendEmpty()
	profileA.Dictionary().MappingTable().AppendEmpty()
	profileA.Dictionary().LinkTable().AppendEmpty()
	rpA := profileA.ResourceProfiles().AppendEmpty()
	rpA.Resource().Attributes().PutStr("service.name", "service-A")
	rpA.Resource().Attributes().PutStr("deployment", "prod")
	spA := rpA.ScopeProfiles().AppendEmpty()
	pA := spA.Profiles().AppendEmpty()
	pA.PeriodType().SetTypeStrindex(1)
	pA.PeriodType().SetUnitStrindex(2)

	profileB := NewProfiles()
	profileB.Dictionary().StringTable().Append("")
	profileB.Dictionary().StringTable().Append("memory")
	profileB.Dictionary().StringTable().Append("bytes")
	profileB.Dictionary().AttributeTable().AppendEmpty()
	profileB.Dictionary().StackTable().AppendEmpty()
	profileB.Dictionary().LocationTable().AppendEmpty()
	profileB.Dictionary().FunctionTable().AppendEmpty()
	profileB.Dictionary().MappingTable().AppendEmpty()
	profileB.Dictionary().LinkTable().AppendEmpty()
	rpB := profileB.ResourceProfiles().AppendEmpty()
	rpB.Resource().Attributes().PutStr("host.name", "host-1")
	rpB.Resource().Attributes().PutStr("cluster.name", "us-east")
	spB := rpB.ScopeProfiles().AppendEmpty()
	pB := spB.Profiles().AppendEmpty()
	pB.PeriodType().SetTypeStrindex(1)
	pB.PeriodType().SetUnitStrindex(2)

	// Record original resource attributes.
	origAttrsA := attrMapToStrings(rpA.Resource().Attributes())
	origAttrsB := attrMapToStrings(rpB.Resource().Attributes())

	// Marshal → Unmarshal (simulates Kafka write/read).
	bytesA, err := marshaler.MarshalProfiles(profileA)
	require.NoError(t, err)
	bytesB, err := marshaler.MarshalProfiles(profileB)
	require.NoError(t, err)

	dstProfiles, err := unmarshaler.UnmarshalProfiles(bytesA)
	require.NoError(t, err)
	srcProfiles, err := unmarshaler.UnmarshalProfiles(bytesB)
	require.NoError(t, err)

	// MergeTo — merges src into dst.
	err = srcProfiles.MergeTo(dstProfiles)
	require.NoError(t, err)

	// Re-marshal the merged result.
	mergedBytes, err := marshaler.MarshalProfiles(dstProfiles)
	require.NoError(t, err)

	// Re-unmarshal (simulates consumer reading from Kafka).
	finalProfiles, err := unmarshaler.UnmarshalProfiles(mergedBytes)
	require.NoError(t, err)

	// After the full round-trip, resource attributes must still match.
	require.Equal(t, 2, finalProfiles.ResourceProfiles().Len(),
		"final profiles should have 2 resource profiles")

	finalAttrsA := attrMapToStrings(finalProfiles.ResourceProfiles().At(0).Resource().Attributes())
	finalAttrsB := attrMapToStrings(finalProfiles.ResourceProfiles().At(1).Resource().Attributes())

	assert.Equal(t, origAttrsA, finalAttrsA,
		"destination resource attributes corrupted after round-trip")
	assert.Equal(t, origAttrsB, finalAttrsB,
		"source resource attributes corrupted after round-trip")
}

func attrMapToStrings(m pcommon.Map) map[string]string {
	result := make(map[string]string, m.Len())
	m.Range(func(k string, v pcommon.Value) bool {
		result[k] = v.AsString()
		return true
	})
	return result
}
