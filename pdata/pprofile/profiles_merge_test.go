// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile

import (
	"fmt"
	"math/rand/v2"
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

// newConformantProfiles returns a spec-conformant Profiles with the required
// index-0 sentinel entry in every dictionary table, so that an error injected
// into a single table is the only invalid reference during a merge.
func newConformantProfiles() Profiles {
	p := NewProfiles()
	d := p.Dictionary()
	d.StringTable().Append("")
	d.AttributeTable().AppendEmpty()
	d.StackTable().AppendEmpty()
	d.LocationTable().AppendEmpty()
	d.FunctionTable().AppendEmpty()
	d.MappingTable().AppendEmpty()
	d.LinkTable().AppendEmpty()
	return p
}

// TestProfilesMergeTo_SwitchDictionaryErrors exercises each error-propagation
// path in the switchDictionary chain by injecting a single out-of-range index
// into an otherwise spec-conformant source. Every case asserts the wrapped
// error surfaces and that the destination is left untouched.
func TestProfilesMergeTo_SwitchDictionaryErrors(t *testing.T) {
	const badIdx = 99

	for _, tt := range []struct {
		name    string
		mutate  func(Profiles)
		wantErr string
	}{
		{
			name: "dictionary attribute key index",
			mutate: func(p Profiles) {
				p.Dictionary().AttributeTable().AppendEmpty().SetKeyStrindex(badIdx)
			},
			wantErr: "invalid key index",
		},
		{
			name: "dictionary function name index",
			mutate: func(p Profiles) {
				p.Dictionary().FunctionTable().AppendEmpty().SetNameStrindex(badIdx)
			},
			wantErr: "invalid name index",
		},
		{
			name: "dictionary mapping filename index",
			mutate: func(p Profiles) {
				p.Dictionary().MappingTable().AppendEmpty().SetFilenameStrindex(badIdx)
			},
			wantErr: "invalid filename index",
		},
		{
			name: "dictionary stack location index",
			mutate: func(p Profiles) {
				p.Dictionary().StackTable().AppendEmpty().LocationIndices().Append(badIdx)
			},
			wantErr: "invalid location index",
		},
		{
			name: "dictionary location line function index",
			mutate: func(p Profiles) {
				loc := p.Dictionary().LocationTable().AppendEmpty()
				loc.Lines().AppendEmpty().SetFunctionIndex(badIdx)
			},
			wantErr: "invalid function index",
		},
		{
			name: "sample attribute index",
			mutate: func(p Profiles) {
				s := p.ResourceProfiles().AppendEmpty().
					ScopeProfiles().AppendEmpty().
					Profiles().AppendEmpty().
					Samples().AppendEmpty()
				s.AttributeIndices().Append(badIdx)
			},
			wantErr: "invalid attribute index",
		},
		{
			name: "profile period type index",
			mutate: func(p Profiles) {
				prof := p.ResourceProfiles().AppendEmpty().
					ScopeProfiles().AppendEmpty().
					Profiles().AppendEmpty()
				prof.PeriodType().SetTypeStrindex(badIdx)
			},
			wantErr: "error switching dictionary for period type",
		},
		{
			name: "profile sample type index",
			mutate: func(p Profiles) {
				prof := p.ResourceProfiles().AppendEmpty().
					ScopeProfiles().AppendEmpty().
					Profiles().AppendEmpty()
				prof.SampleType().SetTypeStrindex(badIdx)
			},
			wantErr: "error switching dictionary for sample type",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := newConformantProfiles()
			tt.mutate(src)
			dest := newConformantProfiles()

			err := src.MergeTo(dest)
			require.ErrorContains(t, err, tt.wantErr)
			assert.Equal(t, 0, dest.ResourceProfiles().Len(),
				"destination must be left untouched on error")
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

func TestMergeTo_FullDedupAndRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 0))
	marshaler := &ProtoMarshaler{}
	unmarshaler := &ProtoUnmarshaler{}

	// Step a: build dst and src; capture original resource attrs.
	dst := newRandomProfiles(r, "dst", 6)
	src := newRandomProfiles(r, "src", 6)
	origDstAttrs := collectResourceAttrs(dst)
	origSrcAttrs := collectResourceAttrs(src)

	// Step b: marshal+unmarshal BOTH to make KeyStrindex references "sticky"
	// (this is the exact pre-condition for the #15084 corruption sequence).
	dstBytes, err := marshaler.MarshalProfiles(dst)
	require.NoError(t, err)
	dst, err = unmarshaler.UnmarshalProfiles(dstBytes)
	require.NoError(t, err)
	require.Equal(t, origDstAttrs, collectResourceAttrs(dst),
		"dst resource attrs must survive initial round-trip (sanity)")
	assertSampleDictAttr(t, dst, "dst-attr-key", "dict-attr-dst")

	srcBytes, err := marshaler.MarshalProfiles(src)
	require.NoError(t, err)
	src, err = unmarshaler.UnmarshalProfiles(srcBytes)
	require.NoError(t, err)
	require.Equal(t, origSrcAttrs, collectResourceAttrs(src),
		"src resource attrs must survive initial round-trip (sanity)")
	assertSampleDictAttr(t, src, "src-attr-key", "dict-attr-src")

	// Step c: merge.
	require.NoError(t, src.MergeTo(dst))

	// Step d: no-duplicate dict entries + in-memory resource attrs correct.
	assertNoDuplicateDictEntries(t, dst.Dictionary())

	expectedMergedAttrs := append(append([]map[string]string{}, origDstAttrs...), origSrcAttrs...)
	require.Equal(t, expectedMergedAttrs, collectResourceAttrs(dst),
		"resource attributes must survive merge in-memory")

	// Verify sample attribute resolves to the original key/value after merge.
	assertSampleDictAttr(t, dst, "dst-attr-key", "dict-attr-dst")
	assertSampleDictAttr(t, dst, "src-attr-key", "dict-attr-src")

	// Step e: marshal+unmarshal the merged result and verify attrs still match.
	mergedBytes, err := marshaler.MarshalProfiles(dst)
	require.NoError(t, err)
	rt, err := unmarshaler.UnmarshalProfiles(mergedBytes)
	require.NoError(t, err)
	require.Equal(t, expectedMergedAttrs, collectResourceAttrs(rt),
		"resource attributes must survive marshal round-trip after merge")

	// Attribute-table entries must also survive the final round-trip.
	assertSampleDictAttr(t, rt, "dst-attr-key", "dict-attr-dst")
	assertSampleDictAttr(t, rt, "src-attr-key", "dict-attr-src")
}

// assertSampleDictAttr verifies that at least one sample in p contains a
// dictionary AttributeTable entry whose resolved key/value matches wantKey/wantVal.
func assertSampleDictAttr(t *testing.T, p Profiles, wantKey, wantVal string) {
	t.Helper()
	d := p.Dictionary()
	for _, rp := range p.ResourceProfiles().All() {
		for _, sp := range rp.ScopeProfiles().All() {
			for _, prof := range sp.Profiles().All() {
				for _, s := range prof.Samples().All() {
					m, err := FromAttributeIndices(d.AttributeTable(), s, d)
					require.NoError(t, err)
					if v, ok := m.Get(wantKey); ok && v.Str() == wantVal {
						return
					}
				}
			}
		}
	}
	t.Fatalf("no sample found with dict attribute %q=%q", wantKey, wantVal)
}

func newRandomProfiles(r *rand.Rand, prefix string, nStrings int) Profiles {
	p := NewProfiles()
	d := p.Dictionary()
	d.StringTable().Append("")
	for i := 1; i < nStrings; i++ {
		d.StringTable().Append(fmt.Sprintf("%s-%d", prefix, i))
	}

	// Add a prefix-distinct key string for the dictionary-level attribute.
	attrKeyIdx := int32(d.StringTable().Len())
	d.StringTable().Append(prefix + "-attr-key")
	attrValStr := "dict-attr-" + prefix

	fn := d.FunctionTable().AppendEmpty()
	fn.SetNameStrindex(int32(r.IntN(nStrings)))

	mp := d.MappingTable().AppendEmpty()
	mp.SetFilenameStrindex(int32(r.IntN(nStrings)))

	loc := d.LocationTable().AppendEmpty()
	ln := loc.Lines().AppendEmpty()
	ln.SetFunctionIndex(0)
	loc.SetMappingIndex(0)

	// index 0: empty sentinel (required by spec; switchDictionary skips index 0)
	d.LinkTable().AppendEmpty()
	// index 1: real link, identical across dst and src so a broken dedup would
	// append a duplicate that assertNoDuplicateDictEntries would catch.
	lnk := d.LinkTable().AppendEmpty()
	lnk.SetTraceID(pcommon.TraceID([16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xAA, 0xBB, 0xCC, 0xDD, 0xAA, 0xBB, 0xCC, 0xDD, 0xAA, 0xBB, 0xCC, 0xDD}))
	lnk.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	stk := d.StackTable().AppendEmpty()
	stk.LocationIndices().Append(0)

	// Dictionary-level attribute: key is attrKeyIdx, value is attrValStr.
	kv := d.AttributeTable().AppendEmpty()
	kv.SetKeyStrindex(attrKeyIdx)
	kv.Value().SetStr(attrValStr)
	attrIdx := int32(d.AttributeTable().Len() - 1)

	dstID := [16]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	srcID := [16]byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	var pid [16]byte
	if prefix == "dst" {
		pid = dstID
	} else {
		pid = srcID
	}

	rp := p.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr(prefix+".key", prefix+".value")
	sp := rp.ScopeProfiles().AppendEmpty()
	sp.Scope().SetName(prefix + "-scope")
	prof := sp.Profiles().AppendEmpty()
	prof.SetProfileID(ProfileID(pid))
	s := prof.Samples().AppendEmpty()
	s.SetStackIndex(0)
	s.SetLinkIndex(1)
	s.AttributeIndices().Append(attrIdx)
	return p
}

func collectResourceAttrs(p Profiles) []map[string]string {
	out := make([]map[string]string, 0, p.ResourceProfiles().Len())
	for i := 0; i < p.ResourceProfiles().Len(); i++ {
		m := map[string]string{}
		p.ResourceProfiles().At(i).Resource().Attributes().Range(func(k string, v pcommon.Value) bool {
			m[k] = v.AsString()
			return true
		})
		out = append(out, m)
	}
	return out
}

func assertNoDuplicateDictEntries(t *testing.T, d ProfilesDictionary) {
	t.Helper()
	st := d.StringTable()
	seen := map[string]bool{}
	for i := 0; i < st.Len(); i++ {
		require.False(t, seen[st.At(i)], "duplicate string %q at %d", st.At(i), i)
		seen[st.At(i)] = true
	}
	ft := d.FunctionTable()
	for i := 0; i < ft.Len(); i++ {
		for j := i + 1; j < ft.Len(); j++ {
			require.False(t, ft.At(i).Equal(ft.At(j)), "duplicate function %d/%d", i, j)
		}
	}
	lt := d.LocationTable()
	for i := 0; i < lt.Len(); i++ {
		for j := i + 1; j < lt.Len(); j++ {
			require.False(t, lt.At(i).Equal(lt.At(j)), "duplicate location %d/%d", i, j)
		}
	}
	sk := d.StackTable()
	for i := 0; i < sk.Len(); i++ {
		for j := i + 1; j < sk.Len(); j++ {
			require.False(t, sk.At(i).Equal(sk.At(j)), "duplicate stack %d/%d", i, j)
		}
	}
	mt := d.MappingTable()
	for i := 0; i < mt.Len(); i++ {
		for j := i + 1; j < mt.Len(); j++ {
			require.False(t, mt.At(i).Equal(mt.At(j)), "duplicate mapping %d/%d", i, j)
		}
	}
	at := d.AttributeTable()
	for i := 0; i < at.Len(); i++ {
		for j := i + 1; j < at.Len(); j++ {
			require.False(t, at.At(i).Equal(at.At(j)), "duplicate attribute %d/%d", i, j)
		}
	}
	lnk := d.LinkTable()
	for i := 0; i < lnk.Len(); i++ {
		for j := i + 1; j < lnk.Len(); j++ {
			require.False(t, lnk.At(i).Equal(lnk.At(j)), "duplicate link %d/%d", i, j)
		}
	}
}
