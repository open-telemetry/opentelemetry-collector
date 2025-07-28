// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"slices"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pprofile.Profiles to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalProfiles to the OTLP/JSON format.
func (*JSONMarshaler) MarshalProfiles(pd Profiles) ([]byte, error) {
	dest := json.BorrowStream(nil)
	defer json.ReturnStream(dest)
	pd.marshalJSONStream(dest)
	return slices.Clone(dest.Buffer()), dest.Error()
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pprofile.Profiles.
type JSONUnmarshaler struct{}

// UnmarshalProfiles from OTLP/JSON format into pprofile.Profiles.
func (*JSONUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	iter := json.BorrowIterator(buf)
	defer json.ReturnIterator(iter)
	td := NewProfiles()
	td.unmarshalJSONIter(iter)
	if iter.Error() != nil {
		return Profiles{}, iter.Error()
	}
	otlp.MigrateProfiles(td.getOrig().ResourceProfiles)
	return td, nil
}

func (ms ResourceProfiles) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "resource":
			internal.UnmarshalJSONIterResource(internal.NewResource(&ms.orig.Resource, ms.state), iter)
		case "scopeProfiles", "scope_profiles":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.ScopeProfiles().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			ms.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms ProfilesDictionary) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "mappingTable", "mapping_table":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.MappingTable().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "locationTable", "location_table":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.LocationTable().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "functionTable", "function_table":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.FunctionTable().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "linkTable", "link_table":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.LinkTable().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "stringTable", "string_table":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.StringTable().Append(iter.ReadString())
				return true
			})
		case "attributeTable", "attribute_table":
			internal.UnmarshalJSONIterMap(internal.NewMap(&ms.orig.AttributeTable, ms.state), iter)
		case "attributeUnits", "attribute_units":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.AttributeUnits().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

// unmarshalJSONIter is not yet used, only here for tests.
func (ms Attribute) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "key":
			ms.orig.Key = iter.ReadString()
		case "value":
			internal.UnmarshalJSONIterValue(internal.NewValue(&ms.orig.Value, ms.state), iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (sp ScopeProfiles) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "scope":
			internal.UnmarshalJSONIterInstrumentationScope(internal.NewInstrumentationScope(&sp.orig.Scope, sp.state), iter)
		case "profiles":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				sp.Profiles().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			sp.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Profile) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "profileId", "profile_id":
			ms.orig.ProfileId.UnmarshalJSONIter(iter)
		case "sampleType", "sample_type":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.SampleType().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "sample":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.Sample().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "locationIndices", "location_indices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.LocationIndices().Append(iter.ReadInt32())
				return true
			})
		case "timeNanos", "time_nanos":
			ms.orig.TimeNanos = iter.ReadInt64()
		case "durationNanos", "duration_nanos":
			ms.orig.DurationNanos = iter.ReadInt64()
		case "periodType", "period_type":
			ms.PeriodType().unmarshalJSONIter(iter)
		case "period":
			ms.orig.Period = iter.ReadInt64()
		case "commentStrindices", "comment_strindices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.CommentStrindices().Append(iter.ReadInt32())
				return true
			})
		case "defaultSampleTypeIndex", "default_sample_type_index":
			ms.orig.DefaultSampleTypeIndex = iter.ReadInt32()
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				ms.AttributeIndices().Append(iter.ReadInt32())
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			ms.orig.DroppedAttributesCount = iter.ReadUint32()
		case "originalPayloadFormat", "original_payload_format":
			ms.orig.OriginalPayloadFormat = iter.ReadString()
		case "originalPayload", "original_payload":
			internal.UnmarshalJSONIterByteSlice(internal.NewByteSlice(&ms.orig.OriginalPayload, ms.state), iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (vt ValueType) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "typeStrindex", "type_strindex":
			vt.orig.TypeStrindex = iter.ReadInt32()
		case "unitStrindex", "unit_strindex":
			vt.orig.UnitStrindex = iter.ReadInt32()
		case "aggregationTemporality", "aggregation_temporality":
			vt.orig.AggregationTemporality = otlpprofiles.AggregationTemporality(iter.ReadInt32())
		default:
			iter.Skip()
		}
		return true
	})
}

func (st Sample) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "locationsStartIndex", "locations_start_index":
			st.orig.LocationsStartIndex = iter.ReadInt32()
		case "locationsLength", "locations_length":
			st.orig.LocationsLength = iter.ReadInt32()
		case "value":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				st.Value().Append(iter.ReadInt64())
				return true
			})
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				st.AttributeIndices().Append(iter.ReadInt32())
				return true
			})
		case "linkIndex", "link_index":
			st.orig.LinkIndex_ = &otlpprofiles.Sample_LinkIndex{LinkIndex: iter.ReadInt32()}
		case "timestampsUnixNano", "timestamps_unix_nano":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				st.TimestampsUnixNano().Append(iter.ReadUint64())
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (m Mapping) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "memoryStart", "memory_start":
			m.orig.MemoryStart = iter.ReadUint64()
		case "memoryLimit", "memory_limit":
			m.orig.MemoryLimit = iter.ReadUint64()
		case "fileOffset", "file_offset":
			m.orig.FileOffset = iter.ReadUint64()
		case "filenameStrindex", "filename_strindex":
			m.orig.FilenameStrindex = iter.ReadInt32()
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				m.AttributeIndices().Append(iter.ReadInt32())
				return true
			})
		case "hasFunctions", "has_functions":
			m.orig.HasFunctions = iter.ReadBool()
		case "hasFilenames", "has_filenames":
			m.orig.HasFilenames = iter.ReadBool()
		case "hasLineNumbers", "has_line_numbers":
			m.orig.HasLineNumbers = iter.ReadBool()
		case "hasInlineFrames", "has_inline_frames":
			m.orig.HasInlineFrames = iter.ReadBool()
		default:
			iter.Skip()
		}
		return true
	})
}

func (l Location) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "mappingIndex", "mapping_index":
			l.orig.MappingIndex_ = &otlpprofiles.Location_MappingIndex{MappingIndex: iter.ReadInt32()}
		case "address":
			l.orig.Address = iter.ReadUint64()
		case "line":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				l.Line().AppendEmpty().unmarshalJSONIter(iter)
				return true
			})
		case "isFolded", "is_folded":
			l.orig.IsFolded = iter.ReadBool()
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *json.Iterator) bool {
				l.AttributeIndices().Append(iter.ReadInt32())
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (l Line) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "functionIndex", "function_index":
			l.orig.FunctionIndex = iter.ReadInt32()
		case "line":
			l.orig.Line = iter.ReadInt64()
		case "column":
			l.orig.Column = iter.ReadInt64()
		default:
			iter.Skip()
		}
		return true
	})
}

func (fn Function) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "nameStrindex", "name_strindex":
			fn.orig.NameStrindex = iter.ReadInt32()
		case "systemNameStrindex", "system_name_strindex":
			fn.orig.SystemNameStrindex = iter.ReadInt32()
		case "filenameStrindex", "filename_strindex":
			fn.orig.FilenameStrindex = iter.ReadInt32()
		case "startLine", "start_line":
			fn.orig.StartLine = iter.ReadInt64()
		default:
			iter.Skip()
		}
		return true
	})
}

func (at AttributeUnit) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "attributeKeyStrindex", "attribute_key_strindex":
			at.orig.AttributeKeyStrindex = iter.ReadInt32()
		case "unitStrindex", "unit_strindex":
			at.orig.UnitStrindex = iter.ReadInt32()
		default:
			iter.Skip()
		}
		return true
	})
}

func (ms Link) unmarshalJSONIter(iter *json.Iterator) {
	iter.ReadObjectCB(func(iter *json.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			ms.orig.TraceId.UnmarshalJSONIter(iter)
		case "spanId", "span_id":
			ms.orig.SpanId.UnmarshalJSONIter(iter)
		default:
			iter.Skip()
		}
		return true
	})
}
