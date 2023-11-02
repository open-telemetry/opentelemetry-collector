// TODO(@petethepig): This file is here temporarily and should be deleted before we merge profiles spec.

package pprofile

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jzelinskie/must"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	go_opentelemetry_io_collector_pdata_internal_data "go.opentelemetry.io/collector/pdata/internal/data"

	common "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"

	otlpprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/denormalized"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprof"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprofextended"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/arrays"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/normalized"
	resource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
)

func prettyPrint(v any) {
	fmt.Println(prettyPrintBuf(v))
}

func prettyPrintBuf(v any) string {
	// have to convert it to json first, otherwise it's too verbose and names are wrong
	str := string(must.NotError(json.MarshalIndent(v, "", "  ")))
	var v2 map[string]any
	json.Unmarshal([]byte(str), &v2)

	b := &bytes.Buffer{}
	yamlEncoder := yaml.NewEncoder(b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(v2)
	return string(b.Bytes())
}

type stringTableBuilder struct {
	originalStringTable []string
	newStringTable      []string
	conversionTable     map[int64]uint32
}

func newStringTableBuilder(originalStringTable []string) *stringTableBuilder {
	return &stringTableBuilder{
		originalStringTable: originalStringTable,
		newStringTable:      []string{""},
		conversionTable:     make(map[int64]uint32),
	}
}

// resolves a string but does not add it to the string table
func (stb *stringTableBuilder) resolveString(n int64) string {
	return stb.originalStringTable[n]
}

func (stb *stringTableBuilder) add(s string) uint32 {
	for i, v := range stb.newStringTable {
		if v == s {
			return uint32(i)
		}
	}
	stb.newStringTable = append(stb.newStringTable, s)
	return uint32(len(stb.newStringTable) - 1)
}

func (stb *stringTableBuilder) convertStringIndex64(n int64) int64 {
	return int64(stb.convertStringIndex(n))
}

func (stb *stringTableBuilder) convertStringIndex(n int64) uint32 {
	if n == 0 {
		return 0
	}
	if ret, ok := stb.conversionTable[n]; ok {
		return ret
	}
	str := stb.originalStringTable[n]
	stb.newStringTable = append(stb.newStringTable, str)
	stringTableID := uint32(len(stb.newStringTable) - 1)
	stb.conversionTable[n] = stringTableID
	return stringTableID
}

type bytesTableBuilder struct {
	originalStringTable []string
	newBytesTable       [][]byte
	conversionTable     map[int64]uint32
}

func newBytesTableBuilder(originalStringTable []string) *bytesTableBuilder {
	return &bytesTableBuilder{
		originalStringTable: originalStringTable,
		newBytesTable:       [][]byte{{}},
		conversionTable:     make(map[int64]uint32),
	}
}

// resolves a bytes but does not add it to the bytes table
func (stb *bytesTableBuilder) resolveBytes(n int64) []byte {
	return []byte(stb.originalStringTable[n])
}

func (stb *bytesTableBuilder) add(s []byte) uint32 {
	for i, v := range stb.newBytesTable {
		if bytes.Equal(v, s) {
			return uint32(i)
		}
	}
	stb.newBytesTable = append(stb.newBytesTable, s)
	return uint32(len(stb.newBytesTable) - 1)
}

func (stb *bytesTableBuilder) convertBytesIndex64(n int64) int64 {
	return int64(stb.convertBytesIndex(n))
}

func (stb *bytesTableBuilder) convertBytesIndex(n int64) uint32 {
	if n == 0 {
		return 0
	}
	if ret, ok := stb.conversionTable[n]; ok {
		return ret
	}
	str := stb.originalStringTable[n]
	stb.newBytesTable = append(stb.newBytesTable, []byte(str))
	bytesTableID := uint32(len(stb.newBytesTable) - 1)
	stb.conversionTable[n] = bytesTableID
	return bytesTableID
}

func PprofToOprof(profile []byte, flavor string) Profiles {
	var pprof pprof.Profile
	err := proto.Unmarshal(ungzipIfNeeded(profile), &pprof)
	if err != nil {
		panic(err)
	}
	return pprofStructToOprof(&pprof, profile, flavor)
	// for _, rps := range oprof.ResourceProfiles {

	// }
	// return pprofileotlp.NewExportRequestFromProfiles(oprof)
	// return &pprofileotlp.ExportRequest{}
	// return
}

// func PprofToOprof(profile []byte, flavor string) *otlpcollectorprofile.ExportProfilesServiceRequest {
// 	var pprof pprof.Profile
// 	proto.Unmarshal(profile, &pprof)
// 	return pprofStructToOprof(&pprof, profile, flavor)
// }

var profileId = go_opentelemetry_io_collector_pdata_internal_data.ProfileID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func pprofStructToOprof(pprof *pprof.Profile, op []byte, flavor string) Profiles {
	var otlpProfile *otlpprofile.ProfileContainer
	switch flavor {
	case "pprof":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Pprof{Pprof: pprofToPprofProfile(pprof)},
		}
	case "normalized":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Normalized{Normalized: pprofToNormalizedProfile(pprof)},
		}
	case "arrays":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Arrays{Arrays: pprofToArraysProfile(pprof)},
		}
	case "denormalized":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Denormalized{Denormalized: pprofToDenormalizedProfile(pprof)},
		}
	case "pprofextended":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Pprofextended{Pprofextended: pprofToPprofextendedProfile(pprof, pprofextendedFlavorEmbedded)},
		}
	case "pprofextendedinterned":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Pprofextended{Pprofextended: pprofToPprofextendedProfile(pprof, pprofextendedFlavorInterned)},
		}
	case "pprofextendedlookup":
		otlpProfile = &otlpprofile.ProfileContainer{
			ProfileId:          profileId,
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.ProfileContainer_Pprofextended{Pprofextended: pprofToPprofextendedProfile(pprof, pprofextendedFlavorLookup)},
		}
	default:
		panic("unknown flavor: " + flavor)
	}

	ps := NewProfiles()
	rp := ps.getOrig()
	rp.ResourceProfiles = []*otlpprofile.ResourceProfiles{{
		Resource: resource.Resource{},
		ScopeProfiles: []*otlpprofile.ScopeProfiles{
			{
				Profiles: []*otlpprofile.ProfileContainer{otlpProfile},
			},
		},
		SchemaUrl: "todo",
	}}
	return ps

	// return &otlpcollectorprofile.ExportProfilesServiceRequest{
	// 	ResourceProfiles: []*otlpprofile.ResourceProfiles{{}},
	// }
}

func readPprof(filename string) *pprof.Profile {
	b, err := os.ReadFile(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	b = ungzipIfNeeded(b)

	var p pprof.Profile
	err = proto.Unmarshal(b, &p)
	if err != nil {
		panic(err)
	}
	return &p
}

func limitSamples(p *pprof.Profile, percentage float64) *pprof.Profile {
	if p == nil {
		return nil
	}
	p.Sample = p.Sample[:int(float64(len(p.Sample))*percentage)]
	return p
}

func ungzipIfNeeded(b []byte) []byte {
	if bytes.HasPrefix(b, []byte{0x1f, 0x8b}) {
		b = must.NotError(ioutil.ReadAll(must.NotError(gzip.NewReader(bytes.NewReader(b)))))
	}
	return b
}

func collapsedToPprof(collapsed string) *pprof.Profile {
	p := &pprof.Profile{
		StringTable: []string{""},
	}

	stringToLocation := make(map[int]uint64)
	stringToFunction := make(map[int]uint64)

	getStringID := func(str string) int {
		stringTableID := slices.IndexFunc(p.StringTable, func(s string) bool { return s == str })

		if stringTableID == -1 {
			p.StringTable = append(p.StringTable, string(str))
			stringTableID = len(p.StringTable) - 1
		}
		return stringTableID
	}

	parserCollapsed(collapsed, func(name []byte, val int, timestamp int, labels map[string]string) {

		funcNames := strings.Split(string(name), ";")
		locationIds := []uint64{}
		for _, funcName := range funcNames {
			stringTableID := getStringID(funcName)
			var functionId uint64
			if functionIdMatch, ok := stringToFunction[stringTableID]; ok {
				functionId = functionIdMatch
			} else {
				functionId = uint64(len(p.Function))
				p.Function = append(p.Function, &pprof.Function{
					Id:   functionId,
					Name: int64(stringTableID),
				})
				stringToFunction[stringTableID] = functionId
			}

			var locationId uint64
			if locationIdMatch, ok := stringToLocation[stringTableID]; ok {
				locationId = locationIdMatch
			} else {
				locationId = uint64(len(p.Location))
				p.Location = append(p.Location, &pprof.Location{
					Id: locationId,
					Line: []*pprof.Line{{
						FunctionId: functionId,
						Line:       0,
					}},
				})
				stringToLocation[stringTableID] = locationId
			}
			locationIds = append(locationIds, uint64(locationId))
		}
		var plabels []*pprof.Label
		for k, v := range labels {
			plabels = append(plabels, &pprof.Label{
				Key: int64(getStringID(k)),
				Str: int64(getStringID(v)),
			})
		}
		if timestamp != 0 {
			plabels = append(plabels, &pprof.Label{
				Key: int64(getStringID("__timestamp__")),
				Num: int64(timestamp),
			})
		}
		p.Sample = append(p.Sample, &pprof.Sample{
			LocationId: locationIds,
			Value:      []int64{int64(val)},
			Label:      plabels,
		})
	})
	p.SampleType = append(p.SampleType, &pprof.ValueType{})

	return p
}

func stringToSpanId(s string) go_opentelemetry_io_collector_pdata_internal_data.SpanID {
	var spanId go_opentelemetry_io_collector_pdata_internal_data.SpanID
	b := must.NotError(hex.DecodeString(s[2:]))
	copy(spanId[:], b)
	return spanId
}

func stringToTraceId(s string) go_opentelemetry_io_collector_pdata_internal_data.TraceID {
	var traceId go_opentelemetry_io_collector_pdata_internal_data.TraceID
	b := must.NotError(hex.DecodeString(s[2:]))
	copy(traceId[:], b)
	return traceId
}

func pprofToDenormalizedProfile(pprofProfile *pprof.Profile) *denormalized.Profile {
	p := &denormalized.Profile{
		Samples: make([]*denormalized.Sample, len(pprofProfile.Sample)),
	}

	for i, s := range pprofProfile.Sample {
		values := make([]uint64, len(s.Value))
		for j, v := range s.Value {
			values[j] = uint64(v)
		}

		p.Samples[i] = &denormalized.Sample{
			Stacktrace: &denormalized.Stacktrace{
				Locations: make([]*denormalized.Location, len(s.LocationId)),
			},
			// Links
			// Attributes
			Values: values,
			// TimestampUnixNano
		}

		span_id := ""
		trace_id := ""
		for _, l := range s.Label {
			if pprofProfile.StringTable[l.Key] == "__timestamp__" {
				p.Samples[i].TimestampUnixNano = uint64(l.Num)
				continue
			}
			if pprofProfile.StringTable[l.Key] == "span_id" {
				span_id = pprofProfile.StringTable[l.Str]
				continue
			}
			if pprofProfile.StringTable[l.Key] == "trace_id" {
				trace_id = pprofProfile.StringTable[l.Str]
				continue
			}
			p.Samples[i].Attributes = append(p.Samples[i].Attributes, common.KeyValue{
				Key:   pprofProfile.StringTable[l.Key],
				Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: pprofProfile.StringTable[l.Str]}},
			})
		}
		if span_id != "" && trace_id != "" {
			p.Samples[i].Links = append(p.Samples[i].Links, &denormalized.Link{
				SpanId:  stringToSpanId(span_id),
				TraceId: stringToTraceId(trace_id),
			})
		}

		for j, lid := range s.LocationId {
			if int(lid) >= len(pprofProfile.Location) {
				p.Samples[i].Stacktrace.Locations[j] = &denormalized.Location{}
			} else {
				loc := pprofProfile.Location[lid]
				p.Samples[i].Stacktrace.Locations[j] = &denormalized.Location{
					// Mapping:
					// Address
					Line: make([]*denormalized.Line, len(loc.Line)),
					// IsFolded
				}

				for k, line := range loc.Line {
					// fi := slices.IndexFunc(pprofProfile.Function, func(f *pprof.Function) bool { return f.Id == line.FunctionId })
					// if pprofProfile.Function[]
					fi := line.FunctionId
					var newFunction *denormalized.Function
					if len(pprofProfile.Function) <= int(fi) {
						newFunction = &denormalized.Function{}
					} else {

						oldFunction := pprofProfile.Function[fi]
						newFunction = &denormalized.Function{
							Name:       pprofProfile.StringTable[oldFunction.Name],
							SystemName: pprofProfile.StringTable[oldFunction.SystemName],
							Filename:   pprofProfile.StringTable[oldFunction.Filename],
							StartLine:  uint32(oldFunction.StartLine),
						}
					}
					p.Samples[i].Stacktrace.Locations[j].Line[k] = &denormalized.Line{
						Function: newFunction,
						Line:     line.Line,
					}
				}
			}
		}
	}
	return p
}

type stacktraceLocation struct {
	offset uint64
	length uint64
}

type pprofextendedFlavor int

const (
	pprofextendedFlavorInterned pprofextendedFlavor = iota
	pprofextendedFlavorLookup
	pprofextendedFlavorEmbedded
)

func pprofToPprofextendedProfile(pprofProfile *pprof.Profile, flavor pprofextendedFlavor) *pprofextended.Profile {
	p := &pprofextended.Profile{}
	stb := newStringTableBuilder(pprofProfile.StringTable)
	// btb := newBytesTableBuilder(pprofProfile.StringTable)
	p.Mapping = make([]*pprofextended.Mapping, len(pprofProfile.Mapping))
	for i, m := range pprofProfile.Mapping {
		p.Mapping[i] = &pprofextended.Mapping{
			MemoryStart: m.MemoryStart,
			MemoryLimit: m.MemoryLimit,
			FileOffset:  m.FileOffset,
			Filename:    stb.convertStringIndex64(m.Filename),
			BuildId:     stb.convertStringIndex64(m.BuildId),
		}
	}
	p.Function = make([]*pprofextended.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Function[i] = &pprofextended.Function{
			Name:       stb.convertStringIndex64(f.Name),
			SystemName: stb.convertStringIndex64(f.SystemName),
			Filename:   stb.convertStringIndex64(f.Filename),
			StartLine:  f.StartLine,
		}
	}
	p.Location = make([]*pprofextended.Location, len(pprofProfile.Location))
	for i, l := range pprofProfile.Location {
		lines := make([]*pprofextended.Line, len(l.Line))
		for j, line := range l.Line {
			lines[j] = &pprofextended.Line{
				FunctionIndex: line.FunctionId,
				Line:          line.Line,
			}
		}
		p.Location[i] = &pprofextended.Location{
			MappingIndex: l.MappingId,
			Address:      l.Address,
			Line:         lines,
		}
	}
	p.Sample = make([]*pprofextended.Sample, len(pprofProfile.Sample))
	// stacktracesMap := make(map[string]stacktraceLocation)
	samples := make([]*pprof.Sample, len(pprofProfile.Sample))
	for i, s := range pprofProfile.Sample {
		samples[i] = s
	}

	sort.Slice(samples, func(i, j int) bool {
		return len(samples[i].LocationId) > len(samples[j].LocationId)
	})

	attributesMap := make(map[string]uint32)

	for i, s := range samples {
		values := make([]int64, len(s.Value))
		for j, v := range s.Value {
			values[j] = int64(v)
		}

		var timestamp uint64

		var span_id string
		var trace_id string
		for _, l := range s.Label {
			keyStr := stb.resolveString(l.Key)
			if keyStr == "__timestamp__" {
				timestamp = uint64(l.Num)
				continue
			} else if keyStr == "span_id" {
				span_id = stb.resolveString(l.Str)
				continue
			} else if keyStr == "trace_id" {
				trace_id = stb.resolveString(l.Str)
				continue
			}
		}

		if span_id != "" && trace_id != "" {
			if flavor == pprofextendedFlavorInterned {
				// attributes = append(attributes, &common.KeyValueInterned{
				// 	Key:   int64(stb.add("trace_id")),
				// 	Value: &common.AnyValueInterned{Value: &common.AnyValueInterned_BytesValue{BytesValue: int64(btb.add([]byte(trace_id)))}},
				// })
				// attributes = append(attributes, &common.KeyValueInterned{
				// 	Key:   int64(stb.add("span_id")),
				// 	Value: &common.AnyValueInterned{Value: &common.AnyValueInterned_BytesValue{BytesValue: int64(btb.add([]byte(span_id)))}},
				// })
			} else if flavor == pprofextendedFlavorEmbedded {
				// attributes2 = append(attributes2, common.KeyValue{
				// 	Key:   "trace_id",
				// 	Value: common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: []byte(trace_id)}},
				// })
				// attributes2 = append(attributes2, common.KeyValue{
				// 	Key:   "span_id",
				// 	Value: common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: []byte(span_id)}},
				// })
			} else if flavor == pprofextendedFlavorLookup {
				aKey := "trace_id" + trace_id
				if _, ok := attributesMap[aKey]; !ok {
					v := stringToTraceId(trace_id)
					p.AttributeTable = append(p.AttributeTable, common.KeyValue{
						Key:   "trace_id",
						Value: common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: v[:]}},
					})
					attributesMap[aKey] = uint32(len(p.AttributeTable) - 1)
				}
				aKey = "span_id" + span_id
				if _, ok := attributesMap[aKey]; !ok {
					v := stringToSpanId(span_id)
					p.AttributeTable = append(p.AttributeTable, common.KeyValue{
						Key:   "span_id",
						Value: common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: v[:]}},
					})
					attributesMap[aKey] = uint32(len(p.AttributeTable) - 1)
				}
			}
		}

		for _, l := range s.Label {
			keyStr := stb.resolveString(l.Key)
			if keyStr == "__timestamp__" {
				continue
			} else if keyStr == "span_id" {
				continue
			} else if keyStr == "trace_id" {
				continue
			}
			// valStr := stb.resolveString(l.Str)
			if flavor == pprofextendedFlavorInterned {
				// attributes = append(attributes, &common.KeyValueInterned{
				// 	Key:   stb.convertStringIndex64(l.Key),
				// 	Value: &common.AnyValueInterned{Value: &common.AnyValueInterned_StringValue{StringValue: stb.convertStringIndex64(l.Str)}},
				// })
			} else if flavor == pprofextendedFlavorEmbedded {
				// attributes2 = append(attributes2, common.KeyValue{
				// 	Key:   stb.resolveString(l.Key),
				// 	Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: stb.resolveString(l.Str)}},
				// })
			} else if flavor == pprofextendedFlavorLookup {
				aKey := keyStr + stb.resolveString(l.Str)
				if _, ok := attributesMap[aKey]; !ok {
					p.AttributeTable = append(p.AttributeTable, common.KeyValue{
						Key:   stb.resolveString(l.Key),
						Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: stb.resolveString(l.Str)}},
					})
					attributesMap[aKey] = uint32(len(p.AttributeTable) - 1)
				}
			}
		}

		var stacktraceStart uint64
		var stacktraceLength uint64
		stacktraceLength = uint64(len(s.LocationId))

		var foundMatch bool

		for i := 0; i < len(p.LocationIndices)-len(s.LocationId); i++ {
			for j := 0; j < len(s.LocationId); j++ {
				if p.LocationIndices[i+j] != int64(s.LocationId[j]) {
					break
				}
				if j == len(s.LocationId)-1 {
					stacktraceStart = uint64(i)
					foundMatch = true
					break
				}
			}
		}

		if !foundMatch {
			stacktraceStart = uint64(len(p.LocationIndices))
			for j := 0; j < len(s.LocationId); j++ {
				p.LocationIndices = append(p.LocationIndices, int64(s.LocationId[j]))
			}
		}

		p.Sample[i] = &pprofextended.Sample{
			Value:               values,
			LocationsStartIndex: stacktraceStart,
			LocationsEndIndex:   stacktraceLength,
		}

		if flavor == pprofextendedFlavorInterned {
			// p.Sample[i].Attributes = attributes
		} else if flavor == pprofextendedFlavorEmbedded {
			// p.Sample[i].Attributes2 = attributes2
		} else if flavor == pprofextendedFlavorLookup {
			// already done above
		}

		if timestamp != 0 {
			p.Sample[i].Timestamps = []uint64{timestamp}
		}
	}

	p.StringTable = stb.newStringTable
	// p.BytesTable = btb.newBytesTable
	return p
}

func pprofToPprofProfile(pprofProfile *pprof.Profile) *pprof.Profile {
	p := &pprof.Profile{}
	stb := newStringTableBuilder(pprofProfile.StringTable)
	p.Mapping = make([]*pprof.Mapping, len(pprofProfile.Mapping))
	for i, m := range pprofProfile.Mapping {
		p.Mapping[i] = &pprof.Mapping{
			Id:              m.Id,
			HasFunctions:    m.HasFunctions,
			HasFilenames:    m.HasFilenames,
			HasLineNumbers:  m.HasLineNumbers,
			HasInlineFrames: m.HasInlineFrames,
			MemoryStart:     m.MemoryStart,
			MemoryLimit:     m.MemoryLimit,
			FileOffset:      m.FileOffset,
			Filename:        stb.convertStringIndex64(m.Filename),
			BuildId:         stb.convertStringIndex64(m.BuildId),
		}
	}
	p.Function = make([]*pprof.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Function[i] = &pprof.Function{
			Id:         f.Id,
			Name:       stb.convertStringIndex64(f.Name),
			SystemName: stb.convertStringIndex64(f.SystemName),
			Filename:   stb.convertStringIndex64(f.Filename),
			StartLine:  f.StartLine,
		}
	}
	p.Location = make([]*pprof.Location, len(pprofProfile.Location))
	for i, l := range pprofProfile.Location {
		lines := make([]*pprof.Line, len(l.Line))
		for j, line := range l.Line {
			lines[j] = &pprof.Line{
				FunctionId: line.FunctionId,
				Line:       line.Line,
			}
		}
		p.Location[i] = &pprof.Location{
			Id:        l.Id,
			MappingId: l.MappingId,
			Address:   l.Address,
			Line:      lines,
			IsFolded:  l.IsFolded,
		}
	}
	p.Sample = make([]*pprof.Sample, len(pprofProfile.Sample))
	for i, s := range pprofProfile.Sample {
		values := make([]int64, len(s.Value))
		for j, v := range s.Value {
			values[j] = int64(v)
		}

		labels := make([]*pprof.Label, len(s.Label))

		for j, l := range s.Label {
			labels[j] = &pprof.Label{
				Key: l.Key,
				Str: stb.convertStringIndex64(l.Str),
				Num: l.Num,
			}
		}

		p.Sample[i] = &pprof.Sample{
			LocationId: s.LocationId,
			Value:      values,
			Label:      labels,
		}
	}

	p.StringTable = stb.newStringTable

	// not copying these to make comparisons to other alternative implementations more relevant
	//   (we don't really use these in other alternative implementations)
	// p.SampleType = pprofProfile.SampleType
	// p.DropFrames = pprofProfile.DropFrames
	// p.KeepFrames = pprofProfile.KeepFrames
	// p.TimeNanos = pprofProfile.TimeNanos
	// p.DurationNanos = pprofProfile.DurationNanos
	// p.PeriodType = pprofProfile.PeriodType
	// p.Period = pprofProfile.Period
	// p.Comment = pprofProfile.Comment
	// p.DefaultSampleType = pprofProfile.DefaultSampleType

	// fmt.Println("original:")
	// prettyPrintBuf(pprofProfile)
	// fmt.Println("modified:")
	// prettyPrintBuf(p)
	return p
}

func pprofToNormalizedProfile(pprofProfile *pprof.Profile) *normalized.Profile {
	p := &normalized.Profile{}

	stb := newStringTableBuilder(pprofProfile.StringTable)
	stacktracesMap := make(map[string]uint64)

	labelsMap := make(map[string]uint32)

	p.Mappings = make([]*normalized.Mapping, len(pprofProfile.Mapping))
	for i, m := range pprofProfile.Mapping {
		p.Mappings[i] = &normalized.Mapping{
			MemoryStart:   m.MemoryStart,
			MemoryLimit:   m.MemoryLimit,
			FileOffset:    m.FileOffset,
			FilenameIndex: stb.convertStringIndex(m.Filename),
			BuildIdIndex:  stb.convertStringIndex(m.BuildId),
		}
	}

	p.Functions = make([]*normalized.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Functions[i] = &normalized.Function{
			NameIndex:       stb.convertStringIndex(f.Name),
			SystemNameIndex: stb.convertStringIndex(f.SystemName),
			FilenameIndex:   stb.convertStringIndex(f.Filename),
			StartLine:       uint32(f.StartLine),
		}
	}
	p.Locations = make([]*normalized.Location, len(pprofProfile.Location))
	for i, l := range pprofProfile.Location {
		lines := make([]*normalized.Line, len(l.Line))
		for j, line := range l.Line {
			lines[j] = &normalized.Line{
				FunctionIndex: uint32(line.FunctionId),
				Line:          uint32(line.Line),
			}
		}
		p.Locations[i] = &normalized.Location{
			MappingIndex: uint32(l.MappingId),
			Address:      l.Address,
			Line:         lines,
		}
	}
	p.Stacktraces = make([]*normalized.Stacktrace, 0)
	p.Samples = make([]*normalized.Sample, len(pprofProfile.Sample))
	for i, s := range pprofProfile.Sample {
		values := make([]int64, len(s.Value))
		for j, v := range s.Value {
			values[j] = int64(v)
		}

		labelsKey := ""
		var links []*normalized.Link
		var timestamp uint64

		var span_id string
		var trace_id string
		for _, l := range s.Label {
			keyStr := stb.resolveString(l.Key)
			if keyStr == "__timestamp__" {
				timestamp = uint64(l.Num)
				continue
			} else if keyStr == "span_id" {
				span_id = stb.resolveString(l.Str)
				continue
			} else if keyStr == "trace_id" {
				trace_id = stb.resolveString(l.Str)
				continue
			}
			labelsKey += fmt.Sprintf("%d=%d,", l.Key, l.Str)
		}

		if span_id != "" && trace_id != "" {
			links = append(links, &normalized.Link{
				SpanId:  stringToSpanId(span_id),
				TraceId: stringToTraceId(trace_id),
			})
		}

		var attributeSetId uint32
		if v, ok := labelsMap[labelsKey]; ok {
			attributeSetId = uint32(v)
		} else {
			as := &normalized.AttributeSet{}
			for _, l := range s.Label {
				keyStr := stb.resolveString(l.Key)
				if keyStr == "__timestamp__" {
					continue
				} else if keyStr == "span_id" {
					continue
				} else if keyStr == "trace_id" {
					continue
				}
				valStr := stb.resolveString(l.Str)
				as.Attributes = append(as.Attributes, common.KeyValue{
					Key:   keyStr,
					Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: valStr}},
				})
			}
			attributeSetId = uint32(len(p.AttributeSets))
			p.AttributeSets = append(p.AttributeSets, as)
			labelsMap[labelsKey] = attributeSetId
		}

		var stacktracesKey string
		for j := 0; j < len(s.LocationId); j++ {
			stacktracesKey += strconv.FormatUint(uint64(s.LocationId[j]), 10) + ","
		}

		var stacktraceIndex uint64

		if v, ok := stacktracesMap[stacktracesKey]; ok {
			stacktraceIndex = v
		} else {
			stacktraceIndex = uint64(len(p.Stacktraces))
			stacktracesMap[stacktracesKey] = stacktraceIndex
			p.Stacktraces = append(p.Stacktraces, &normalized.Stacktrace{
				LocationIndices: make([]uint32, len(s.LocationId)),
			})
			for j := 0; j < len(s.LocationId); j++ {
				p.Stacktraces[stacktraceIndex].LocationIndices[j] = uint32(s.LocationId[j])
			}
		}

		// p.Stacktraces[stacktraceIndex] = &normalized.Stacktrace{
		// 	LocationIndices: make([]uint32, len(s.LocationId)),
		// }

		// for j := 0; j < len(s.LocationId); j++ {
		// 	p.Stacktraces[stacktraceIndex].LocationIndices[j] = uint32(s.LocationId[j])
		// }

		p.Samples[i] = &normalized.Sample{
			Values:              values,
			StacktraceIndex:     uint32(stacktraceIndex),
			LinkIndices:         []uint32{}, // TODO
			AttributeSetIndices: []uint32{attributeSetId},
			TimestampUnixNano:   timestamp,
		}
	}
	p.StringTable = stb.newStringTable
	return p
}

func pprofToArraysProfile(pprofProfile *pprof.Profile) *arrays.Profile {
	p := &arrays.Profile{}
	stb := newStringTableBuilder(pprofProfile.StringTable)
	stacktracesMap := make(map[string]uint64)

	labelsMap := make(map[string]uint32)

	p.Functions = make([]*arrays.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Functions[i] = &arrays.Function{
			NameIndex:       stb.convertStringIndex(f.Name),
			SystemNameIndex: stb.convertStringIndex(f.SystemName),
			FilenameIndex:   stb.convertStringIndex(f.Filename),
			StartLine:       uint32(f.StartLine),
		}
	}

	p.Mappings = make([]*arrays.Mapping, len(pprofProfile.Mapping))
	for i, m := range pprofProfile.Mapping {
		p.Mappings[i] = &arrays.Mapping{
			MemoryStart:   m.MemoryStart,
			MemoryLimit:   m.MemoryLimit,
			FileOffset:    m.FileOffset,
			FilenameIndex: stb.convertStringIndex(m.Filename),
			BuildIdIndex:  stb.convertStringIndex(m.BuildId),
			// SymbolicInfo:        m.SymbolicInfo,
			// AttributeSetIndices: m.AttributeSetIndices,
		}
	}
	p.Locations = make([]*arrays.Location, len(pprofProfile.Location))
	for i, l := range pprofProfile.Location {
		lines := make([]*arrays.Line, len(l.Line))
		for j, line := range l.Line {
			lines[j] = &arrays.Line{
				FunctionIndex: uint32(line.FunctionId),
				Line:          uint32(line.Line),
			}
		}
		p.Locations[i] = &arrays.Location{
			MappingIndex: uint32(l.MappingId),
			Address:      l.Address,
			Line:         lines,
		}
	}
	p.Stacktraces = make([]*arrays.Stacktrace, 0)
	p.ProfileTypes = make([]*arrays.ProfileType, len(pprofProfile.SampleType))

	p.AttributeSets = append(p.AttributeSets, &arrays.AttributeSet{})
	p.Links = append(p.Links, &arrays.Link{})

	for k := 0; k < len(pprofProfile.SampleType); k++ {
		p.ProfileTypes[k] = &arrays.ProfileType{}

		p.ProfileTypes[k].Values = make([]int64, len(pprofProfile.Sample))
		p.ProfileTypes[k].StacktraceIndices = make([]uint32, len(pprofProfile.Sample))

		for i, s := range pprofProfile.Sample {
			values := make([]int64, len(s.Value))
			for j, v := range s.Value {
				values[j] = int64(v)
			}

			labelsKey := ""
			var timestamp uint64
			var linkIndex uint32
			var span_id string
			var trace_id string
			for _, l := range s.Label {
				keyStr := stb.resolveString(l.Key)
				if keyStr == "__timestamp__" {
					timestamp = uint64(l.Num)
					continue
				} else if keyStr == "span_id" {
					span_id = stb.resolveString(l.Str)
					continue
				} else if keyStr == "trace_id" {
					trace_id = stb.resolveString(l.Str)
					continue
				}
				labelsKey += fmt.Sprintf("%d=%d,", l.Key, l.Str)
			}

			// TODO: dedup links
			if span_id != "" && trace_id != "" {
				linkIndex = uint32(len(p.Links))
				p.Links = append(p.Links, &arrays.Link{
					TraceId: stringToTraceId(trace_id),
					SpanId:  stringToSpanId(span_id),
				})
			}

			var attributeSetId uint32
			if v, ok := labelsMap[labelsKey]; ok {
				attributeSetId = uint32(v)
			} else {
				as := &arrays.AttributeSet{}
				for _, l := range s.Label {
					keyStr := stb.resolveString(l.Key)
					if keyStr == "__timestamp__" {
						continue
					} else if keyStr == "span_id" {
						continue
					} else if keyStr == "trace_id" {
						continue
					}
					as.Attributes = append(as.Attributes, common.KeyValue{
						Key:   stb.resolveString(l.Key),
						Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: stb.resolveString(l.Str)}},
					})
				}
				attributeSetId = uint32(len(p.AttributeSets))
				p.AttributeSets = append(p.AttributeSets, as)
				labelsMap[labelsKey] = attributeSetId
			}

			var stacktracesKey string
			for j := 0; j < len(s.LocationId); j++ {
				stacktracesKey += strconv.FormatUint(uint64(s.LocationId[j]), 10) + ","
			}

			var stacktraceIndex uint64

			if v, ok := stacktracesMap[stacktracesKey]; ok {
				stacktraceIndex = v
			} else {
				stacktraceIndex = uint64(len(p.Stacktraces))
				stacktracesMap[stacktracesKey] = stacktraceIndex
				p.Stacktraces = append(p.Stacktraces, &arrays.Stacktrace{
					LocationIndices: make([]uint32, len(s.LocationId)),
				})
				for j := 0; j < len(s.LocationId); j++ {
					p.Stacktraces[stacktraceIndex].LocationIndices[j] = uint32(s.LocationId[j])
				}
			}

			p.ProfileTypes[k].Values[i] = values[0]
			p.ProfileTypes[k].StacktraceIndices[i] = uint32(stacktraceIndex)

			if linkIndex > 0 {
				if p.ProfileTypes[k].LinkIndices == nil {
					p.ProfileTypes[k].LinkIndices = make([]uint32, len(pprofProfile.Sample))
				}
				p.ProfileTypes[k].LinkIndices[i] = linkIndex
			}

			if attributeSetId > 0 {
				if p.ProfileTypes[k].AttributeSetIndices == nil {
					p.ProfileTypes[k].AttributeSetIndices = make([]uint32, len(pprofProfile.Sample))
				}
				p.ProfileTypes[k].AttributeSetIndices[i] = attributeSetId
			}

			if timestamp > 0 {
				if p.ProfileTypes[k].Timestamps == nil {
					p.ProfileTypes[k].Timestamps = make([]uint64, len(pprofProfile.Sample))
				}
				p.ProfileTypes[k].Timestamps[i] = timestamp
			}
		}
	}
	p.StringTable = stb.newStringTable
	return p
}

func parserCollapsed(profile string, cb func(name []byte, val, timestamp int, labels map[string]string)) error {
	r := strings.NewReader(profile)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}

		line := scanner.Bytes()

		arr := bytes.Split(line, []byte(" "))

		stacktrace := arr[0]
		var count []byte
		if len(arr) > 1 {
			count = arr[1]
		}
		labels := make(map[string]string)
		var timestamp int
		if len(arr) > 2 {
			for _, x := range arr[2:] {
				if bytes.Contains(x, []byte("=")) {
					labelsArr := bytes.Split(x, []byte(","))
					for _, label := range labelsArr {
						kv := bytes.Split(label, []byte("="))
						labels[string(kv[0])] = string(kv[1])
					}
				} else {
					timestampInt, err := strconv.Atoi(string(x))
					if err != nil {
						continue
						return err
					}
					timestamp = int(timestampInt)
				}
			}
		}

		i, err := strconv.Atoi(string(count))
		if err != nil {
			return err
		}
		cb(stacktrace, i, timestamp, labels)
	}
	return nil
}

func calculateLabelsCount(p *pprof.Profile) (r uint64) {
	labelSets := make(map[string]bool)
	for i := 0; i < len(p.Sample); i++ {
		key := ""
		// TODO: sort?
		for j := 0; j < len(p.Sample[i].Label); j++ {
			key += p.StringTable[p.Sample[i].Label[j].Key]
			str := p.Sample[i].Label[j].Str
			if str == 0 {
				key += fmt.Sprintf("%d", p.Sample[i].Label[j].Num)
			} else {
				key += p.StringTable[str]
			}
		}
		if _, ok := labelSets[key]; !ok {
			r++
			labelSets[key] = true
		}
	}
	return r
}

func gzipBuffer(p []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(p); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func ungzipBuffer(p []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write(p); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func GetLabels(r ResourceProfiles) []common.KeyValue {
	return r.orig.Resource.Attributes
}

func OprofToPprof(pWrapper Profile) []byte {
	p := pWrapper.orig

	dst := pprof.Profile{}
	// orig := p.getOrig()
	// for _, rp := range orig.GetResourceProfiles() {
	// 	for _, sp := range rp.ScopeProfiles {
	// 		for _, p := range sp.Profiles {
	switch p.AlternativeProfile.(type) {
	case *otlpprofile.ProfileContainer_Arrays:
		arrays := p.AlternativeProfile.(*otlpprofile.ProfileContainer_Arrays)
		src := arrays.Arrays
		dst.StringTable = src.StringTable
		dst.Function = make([]*pprof.Function, len(src.Functions))
		for i, f := range src.Functions {
			dst.Function[i] = &pprof.Function{
				Id:         uint64(i + 1),
				Name:       int64(f.NameIndex),
				SystemName: int64(f.SystemNameIndex),
				Filename:   int64(f.FilenameIndex),
				StartLine:  int64(f.StartLine),
			}
		}
		dst.Location = make([]*pprof.Location, len(src.Locations))
		for i, l := range src.Locations {
			lines := make([]*pprof.Line, len(l.Line))
			for j, line := range l.Line {
				lines[j] = &pprof.Line{
					FunctionId: uint64(line.FunctionIndex),
					Line:       int64(line.Line),
				}
			}
			dst.Location[i] = &pprof.Location{
				Id:        uint64(i + 1),
				MappingId: uint64(l.MappingIndex),
				Address:   uint64(l.Address),
				Line:      lines,
			}
		}
		dst.Mapping = make([]*pprof.Mapping, len(src.Mappings))
		for i, m := range src.Mappings {
			dst.Mapping[i] = &pprof.Mapping{
				Id:          uint64(i + 1),
				MemoryStart: m.MemoryStart,
				MemoryLimit: m.MemoryLimit,
				FileOffset:  m.FileOffset,
				Filename:    int64(m.FilenameIndex),
				BuildId:     int64(m.BuildIdIndex),
			}
		}

		// TODO: implement these
		dst.TimeNanos = int64(time.Now().UnixNano())
		dst.DurationNanos = 201278458
		dst.Period = 10000000

		// TODO: implement this, dedup with SampleType
		dst.PeriodType = &pprof.ValueType{
			Type: 3,
			Unit: 4,
		}
		dst.SampleType = make([]*pprof.ValueType, len(src.ProfileTypes))
		for i := range src.ProfileTypes {
			// TODO: implement this, dedup with PeriodType
			dst.SampleType[i] = &pprof.ValueType{
				Type: int64(i*2 + 1),
				Unit: int64(i*2 + 2),
			}
		}

		sampleLookup := make(map[uint32]*pprof.Sample)

		for i, pt := range src.ProfileTypes {
			for j, sti := range pt.StacktraceIndices {
				value := pt.Values[j]
				stacktrace := src.Stacktraces[sti]
				var sample *pprof.Sample
				if s, ok := sampleLookup[sti]; ok {
					sample = s
				} else {
					sample = &pprof.Sample{
						Value: make([]int64, len(src.ProfileTypes)),
					}
					dst.Sample = append(dst.Sample, sample)
					sampleLookup[sti] = sample
				}
				sample.Value[i] = value
				sample.LocationId = make([]uint64, len(stacktrace.LocationIndices))
				for k, li := range stacktrace.LocationIndices {
					sample.LocationId[k] = uint64(li)
				}
				// TODO: labels
			}
		}
		// TODO: implement support for Links
		// TODO: implement support for SampleTypes
		// TODO: implement support for Stacktraces
		// TODO: implement support for AttributeSets
	default:
		panic("unsupported profile type")
	}
	return must.NotError(dst.Marshal())
}

func splitIntoManyEventsWithTimestamps(p *pprof.Profile) *pprof.Profile {
	newSamples := []*pprof.Sample{}
	p.StringTable = append(p.StringTable, "__timestamp__")
	for _, s := range p.Sample {
		// only supported for cpu profiles
		for i := 0; i < int(s.Value[0]); i++ {
			newLabels := []*pprof.Label{}
			for _, l := range s.Label {
				newLabels = append(newLabels, l)
			}
			// adds timestamps
			newLabels = append(newLabels, &pprof.Label{
				Key: int64(len(p.StringTable) - 1),
				Num: int64(time.Now().Add(time.Duration(i) * time.Millisecond).Unix()),
			})
			newSamples = append(newSamples, &pprof.Sample{
				Value:      []int64{1, 1},
				LocationId: s.LocationId,
				Label:      newLabels,
			})
		}
	}
	fmt.Println("before:", len(p.Sample))
	p.Sample = newSamples
	fmt.Println("after:", len(p.Sample))
	return p
}
