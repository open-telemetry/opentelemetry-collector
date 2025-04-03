// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"bytes"
	"fmt"

	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpprofiles "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1development"
	"go.opentelemetry.io/collector/pdata/internal/json"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
)

// JSONMarshaler marshals pprofile.Profiles to JSON bytes using the OTLP/JSON format.
type JSONMarshaler struct{}

// MarshalProfiles to the OTLP/JSON format.
func (*JSONMarshaler) MarshalProfiles(td Profiles) ([]byte, error) {
	buf := bytes.Buffer{}
	pb := internal.ProfilesToProto(internal.Profiles(td))
	err := json.Marshal(&buf, &pb)
	return buf.Bytes(), err
}

// JSONUnmarshaler unmarshals OTLP/JSON formatted-bytes to pprofile.Profiles.
type JSONUnmarshaler struct{}

// UnmarshalProfiles from OTLP/JSON format into pprofile.Profiles.
func (*JSONUnmarshaler) UnmarshalProfiles(buf []byte) (Profiles, error) {
	iter := jsoniter.ConfigFastest.BorrowIterator(buf)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	td := NewProfiles()
	td.unmarshalJsoniter(iter)
	if iter.Error != nil {
		return Profiles{}, iter.Error
	}
	otlp.MigrateProfiles(td.getOrig().ResourceProfiles)
	return td, nil
}

func (ms Profiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resourceProfiles", "resource_profiles":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				ms.ResourceProfiles().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (rp ResourceProfiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "resource":
			json.ReadResource(iter, internal.GetOrigResource(internal.Resource(rp.Resource())))
		case "scopeProfiles", "scope_profiles":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				rp.ScopeProfiles().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "schemaUrl", "schema_url":
			rp.orig.SchemaUrl = iter.ReadString()
		default:
			iter.Skip()
		}
		return true
	})
}

func (sp ScopeProfiles) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "scope":
			json.ReadScope(iter, &sp.orig.Scope)
		case "profiles":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				sp.Profiles().AppendEmpty().unmarshalJsoniter(iter)
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

func (p Profile) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "profileId", "profile_id":
			if err := p.orig.ProfileId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("profileContainer.profileId", fmt.Sprintf("parse profile_id:%v", err))
			}
		case "sampleType", "sample_type":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.SampleType().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "sample":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.Sample().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "mappingTable", "mapping_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.MappingTable().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "locationTable", "location_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.LocationTable().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "locationIndices", "location_indices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.LocationIndices().Append(json.ReadInt32(iter))
				return true
			})
		case "functionTable", "function_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.FunctionTable().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "attributeTable", "attribute_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.orig.AttributeTable = append(p.orig.AttributeTable, json.ReadAttribute(iter))
				return true
			})
		case "attributeUnits", "attribute_units":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.AttributeUnits().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "linkTable", "link_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.LinkTable().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "stringTable", "string_table":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.StringTable().Append(iter.ReadString())
				return true
			})
		case "timeNanos", "time_nanos":
			p.orig.TimeNanos = json.ReadInt64(iter)
		case "durationNanos", "duration_nanos":
			p.orig.DurationNanos = json.ReadInt64(iter)
		case "periodType", "period_type":
			p.PeriodType().unmarshalJsoniter(iter)
		case "period":
			p.orig.Period = json.ReadInt64(iter)
		case "commentStrindices", "comment_strindices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.CommentStrindices().Append(json.ReadInt32(iter))
				return true
			})
		case "defaultSampleTypeStrindex", "default_sample_type_strindex":
			p.orig.DefaultSampleTypeStrindex = json.ReadInt32(iter)
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				p.AttributeIndices().Append(json.ReadInt32(iter))
				return true
			})
		case "droppedAttributesCount", "dropped_attributes_count":
			p.orig.DroppedAttributesCount = json.ReadUint32(iter)
		case "originalPayloadFormat", "original_payload_format":
			p.orig.OriginalPayloadFormat = iter.ReadString()
		case "originalPayload", "original_payload":
			p.orig.OriginalPayload = iter.ReadStringAsSlice()
		default:
			iter.Skip()
		}
		return true
	})
}

func (vt ValueType) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "typeStrindex", "type_strindex":
			vt.orig.TypeStrindex = json.ReadInt32(iter)
		case "unitStrindex", "unit_strindex":
			vt.orig.UnitStrindex = json.ReadInt32(iter)
		case "aggregationTemporality", "aggregation_temporality":
			vt.orig.AggregationTemporality = otlpprofiles.AggregationTemporality(json.ReadInt32(iter))
		default:
			iter.Skip()
		}
		return true
	})
}

func (st Sample) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "locationsStartIndex", "locations_start_index":
			st.orig.LocationsStartIndex = json.ReadInt32(iter)
		case "locationsLength", "locations_length":
			st.orig.LocationsLength = json.ReadInt32(iter)
		case "value":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				st.Value().Append(json.ReadInt64(iter))
				return true
			})
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				st.AttributeIndices().Append(json.ReadInt32(iter))
				return true
			})
		case "linkIndex", "link_index":
			st.orig.LinkIndex_ = &otlpprofiles.Sample_LinkIndex{LinkIndex: json.ReadInt32(iter)}
		case "timestampsUnixNano", "timestamps_unix_nano":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				st.TimestampsUnixNano().Append(json.ReadUint64(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (m Mapping) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "memoryStart", "memory_start":
			m.orig.MemoryStart = json.ReadUint64(iter)
		case "memoryLimit", "memory_limit":
			m.orig.MemoryLimit = json.ReadUint64(iter)
		case "fileOffset", "file_offset":
			m.orig.FileOffset = json.ReadUint64(iter)
		case "filenameStrindex", "filename_strindex":
			m.orig.FilenameStrindex = json.ReadInt32(iter)
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				m.AttributeIndices().Append(json.ReadInt32(iter))
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

func (l Location) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "mappingIndex", "mapping_index":
			l.orig.MappingIndex_ = &otlpprofiles.Location_MappingIndex{MappingIndex: json.ReadInt32(iter)}
		case "address":
			l.orig.Address = json.ReadUint64(iter)
		case "line":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				l.Line().AppendEmpty().unmarshalJsoniter(iter)
				return true
			})
		case "isFolded", "is_folded":
			l.orig.IsFolded = iter.ReadBool()
		case "attributeIndices", "attribute_indices":
			iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
				l.AttributeIndices().Append(json.ReadInt32(iter))
				return true
			})
		default:
			iter.Skip()
		}
		return true
	})
}

func (l Line) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "functionIndex", "function_index":
			l.orig.FunctionIndex = json.ReadInt32(iter)
		case "line":
			l.orig.Line = json.ReadInt64(iter)
		case "column":
			l.orig.Column = json.ReadInt64(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (fn Function) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "nameStrindex", "name_strindex":
			fn.orig.NameStrindex = json.ReadInt32(iter)
		case "systemNameStrindex", "system_name_strindex":
			fn.orig.SystemNameStrindex = json.ReadInt32(iter)
		case "filenameStrindex", "filename_strindex":
			fn.orig.FilenameStrindex = json.ReadInt32(iter)
		case "startLine", "start_line":
			fn.orig.StartLine = json.ReadInt64(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (at AttributeUnit) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "attributeKeyStrindex", "attribute_key_strindex":
			at.orig.AttributeKeyStrindex = json.ReadInt32(iter)
		case "unitStrindex", "unit_strindex":
			at.orig.UnitStrindex = json.ReadInt32(iter)
		default:
			iter.Skip()
		}
		return true
	})
}

func (l Link) unmarshalJsoniter(iter *jsoniter.Iterator) {
	iter.ReadObjectCB(func(iter *jsoniter.Iterator, f string) bool {
		switch f {
		case "traceId", "trace_id":
			if err := l.orig.TraceId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("link.traceId", fmt.Sprintf("parse trace_id:%v", err))
			}
		case "spanId", "span_id":
			if err := l.orig.SpanId.UnmarshalJSON([]byte(iter.ReadString())); err != nil {
				iter.ReportError("link.spanId", fmt.Sprintf("parse span_id:%v", err))
			}
		default:
			iter.Skip()
		}
		return true
	})
}
