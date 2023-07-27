// TODO(@petethepig): This file is here temporarily and should be deleted before we merge profiles spec.

package pprofile

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jzelinskie/must"
	"golang.org/x/exp/slices"

	common "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"

	otlpprofile "go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/denormalized"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/alternatives/pprof"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/arrays"
	"go.opentelemetry.io/collector/pdata/internal/data/protogen/profiles/v1/normalized"
	resource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"
)

func collapsedToOprof(profile string, flavor string) Profiles {
	pprof := collapsedToPprof(profile)
	return pprofStructToOprof(pprof, []byte(profile), flavor)
}

func prettyPrint(v any) {
	str := string(must.NotError(json.MarshalIndent(v, "", "  ")))
	// if len(str) > 5000 {
	// 	str = str[:5000]
	// }
	fmt.Println(str)
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

func pprofStructToOprof(pprof *pprof.Profile, op []byte, flavor string) Profiles {
	var otlpProfile *otlpprofile.Profile
	switch flavor {
	case "pprof":
		otlpProfile = &otlpprofile.Profile{
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.Profile_Pprof{Pprof: pprof},
		}
	case "normalized":
		otlpProfile = &otlpprofile.Profile{
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.Profile_Normalized{Normalized: pprofToNormalizedProfile(pprof)},
		}
	case "arrays":
		otlpProfile = &otlpprofile.Profile{
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.Profile_Arrays{Arrays: pprofToArraysProfile(pprof)},
		}
	case "denormalized":
		otlpProfile = &otlpprofile.Profile{
			OriginalPayload:    op,
			AlternativeProfile: &otlpprofile.Profile_Denormalized{Denormalized: pprofToDenormalizedProfile(pprof)},
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
				Profiles: []*otlpprofile.Profile{otlpProfile},
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
	b := must.NotError(ioutil.ReadFile(filename))
	b = ungzipIfNeeded(b)

	var p pprof.Profile
	err := proto.Unmarshal(b, &p)
	if err != nil {
		panic(err)
	}
	return &p
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

func pprofToNormalizedProfile(pprofProfile *pprof.Profile) *normalized.Profile {
	p := &normalized.Profile{
		StringTable: pprofProfile.StringTable,
	}

	labelsMap := make(map[string]uint32)

	p.Functions = make([]*normalized.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Functions[i] = &normalized.Function{
			NameIndex:       uint32(f.Name),
			SystemNameIndex: uint32(f.SystemName),
			FilenameIndex:   uint32(f.Filename),
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
	p.Stacktraces = make([]*normalized.Stacktrace, len(pprofProfile.Sample))
	p.Samples = make([]*normalized.Sample, len(pprofProfile.Sample))
	for i, s := range pprofProfile.Sample {
		values := make([]int64, len(s.Value))
		for j, v := range s.Value {
			values[j] = int64(v)
		}

		labelsKey := ""
		var timestamp uint64
		for _, l := range s.Label {
			if p.StringTable[l.Key] == "__timestamp__" {
				timestamp = uint64(l.Num)
			}
			labelsKey += fmt.Sprintf("%d=%d,", l.Key, l.Str)
		}

		var attributeSetId uint32
		if v, ok := labelsMap[labelsKey]; ok {
			attributeSetId = uint32(v)
		} else {
			as := &normalized.AttributeSet{}
			for _, l := range s.Label {
				if p.StringTable[l.Key] == "__timestamp__" {
					continue
				}
				// l.Key = uint64(len(p.StringTable))
				as.Attributes = append(as.Attributes, common.KeyValue{
					Key:   p.StringTable[l.Key],
					Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: p.StringTable[l.Str]}},
				})
			}
			attributeSetId = uint32(len(p.AttributeSets))
			p.AttributeSets = append(p.AttributeSets, as)
			labelsMap[labelsKey] = attributeSetId
		}

		// TODO: implement deduplication for stacktraces?
		p.Stacktraces[i] = &normalized.Stacktrace{
			LocationIndices: make([]uint32, len(s.LocationId)),
		}

		for j := 0; j < len(s.LocationId); j++ {
			p.Stacktraces[i].LocationIndices[j] = uint32(s.LocationId[j])
		}

		p.Samples[i] = &normalized.Sample{
			Values:              values,
			StacktraceIndex:     uint32(i),
			LinkIndices:         []uint32{}, // TODO
			AttributeSetIndices: []uint32{attributeSetId},
			TimestampUnixNano:   timestamp,
		}
	}
	return p
}

func pprofToArraysProfile(pprofProfile *pprof.Profile) *arrays.Profile {

	p := &arrays.Profile{
		StringTable: pprofProfile.StringTable,
	}

	labelsMap := make(map[string]uint32)

	p.Functions = make([]*arrays.Function, len(pprofProfile.Function))
	for i, f := range pprofProfile.Function {
		p.Functions[i] = &arrays.Function{
			NameIndex:       uint32(f.Name),
			SystemNameIndex: uint32(f.SystemName),
			FilenameIndex:   uint32(f.Filename),
			StartLine:       uint32(f.StartLine),
		}
	}

	p.Mappings = make([]*arrays.Mapping, len(pprofProfile.Mapping))
	for i, m := range pprofProfile.Mapping {
		p.Mappings[i] = &arrays.Mapping{
			MemoryStart:   m.MemoryStart,
			MemoryLimit:   m.MemoryLimit,
			FileOffset:    m.FileOffset,
			FilenameIndex: uint32(m.Filename),
			BuildIdIndex:  uint32(m.BuildId),
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
	p.Stacktraces = make([]*arrays.Stacktrace, len(pprofProfile.Sample))
	p.ProfileTypes = make([]*arrays.ProfileType, len(pprofProfile.SampleType))

	for k := 0; k < len(pprofProfile.SampleType); k++ {
		p.ProfileTypes[k] = &arrays.ProfileType{}

		p.ProfileTypes[k].Values = make([]int64, len(pprofProfile.Sample))
		p.ProfileTypes[k].StacktraceIndices = make([]uint32, len(pprofProfile.Sample)) // TODO: drop zeros?
		p.ProfileTypes[k].LinkIndices = make([]uint32, len(pprofProfile.Sample))
		p.ProfileTypes[k].AttributeSetIndices = make([]uint32, len(pprofProfile.Sample))
		p.ProfileTypes[k].Timestamps = make([]uint64, len(pprofProfile.Sample))

		for i, s := range pprofProfile.Sample {
			values := make([]int64, len(s.Value))
			for j, v := range s.Value {
				values[j] = int64(v)
			}

			labelsKey := ""
			var timestamp uint64
			for _, l := range s.Label {
				if p.StringTable[l.Key] == "__timestamp__" {
					timestamp = uint64(l.Num)
				}
				labelsKey += fmt.Sprintf("%d=%d,", l.Key, l.Str)
			}

			var attributeSetId uint32
			if v, ok := labelsMap[labelsKey]; ok {
				attributeSetId = uint32(v)
			} else {
				as := &arrays.AttributeSet{}
				for _, l := range s.Label {
					if p.StringTable[l.Key] == "__timestamp__" {
						continue
					}
					// l.Key = uint64(len(p.StringTable))
					as.Attributes = append(as.Attributes, common.KeyValue{
						Key:   p.StringTable[l.Key],
						Value: common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: p.StringTable[l.Str]}},
					})
				}
				attributeSetId = uint32(len(p.AttributeSets))
				p.AttributeSets = append(p.AttributeSets, as)
				labelsMap[labelsKey] = attributeSetId
			}

			// TODO: implement deduplication for stacktraces?
			if p.Stacktraces[i] == nil {
				p.Stacktraces[i] = &arrays.Stacktrace{
					LocationIndices: make([]uint32, len(s.LocationId)),
				}
				for j := 0; j < len(s.LocationId); j++ {
					p.Stacktraces[i].LocationIndices[j] = uint32(s.LocationId[j])
				}
			}

			// p.Samples[i] = &arrays.Sample{
			// 	Values:              values,
			// 	StacktraceIndex:     uint32(i),
			// 	LinkIndices:         []uint32{}, // TODO
			// 	AttributeSetIndices: []uint32{attributeSetId},
			// 	TimestampUnixNano:   timestamp,
			// }
			p.ProfileTypes[k].Values[i] = values[0]
			p.ProfileTypes[k].StacktraceIndices[i] = uint32(i)
			p.ProfileTypes[k].LinkIndices[i] = 0 // TODO
			p.ProfileTypes[k].AttributeSetIndices[i] = attributeSetId
			p.ProfileTypes[k].Timestamps[i] = timestamp
		}
	}
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
			for _, x := range arr {
				if bytes.Contains(x, []byte("=")) {
					labelsArr := bytes.Split(x, []byte(","))
					for _, label := range labelsArr {
						kv := bytes.Split(label, []byte("="))
						labels[string(kv[0])] = string(kv[1])
					}
				} else {
					timestampInt, err := strconv.Atoi(string(count))
					if err != nil {
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
			key += p.StringTable[p.Sample[i].Label[j].Str]
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
	case *otlpprofile.Profile_Arrays:
		arrays := p.AlternativeProfile.(*otlpprofile.Profile_Arrays)
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
		for i, _ := range src.ProfileTypes {
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
