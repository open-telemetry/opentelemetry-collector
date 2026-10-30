// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import (
	"math"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211

	negZeroBits uint64 = 1 << 63
)

func mixU64(h, v uint64) uint64 {
	h ^= v
	h *= fnvPrime64
	return h
}

// mixBytes reads the slice in place: ByteSlice.AsRaw would copy, and hashing
// must not allocate per entry.
func mixBytes(h uint64, b pcommon.ByteSlice) uint64 {
	for i := 0; i < b.Len(); i++ {
		h = mixU64(h, uint64(b.At(i)))
	}
	return mixU64(h, uint64(b.Len()))
}

func mixString(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h = mixU64(h, uint64(s[i]))
	}
	return mixU64(h, uint64(len(s)))
}

// hashValue mixes an attribute value into h. It must agree with
// [pcommon.Value.Equal]: values that compare equal must hash equal, or the
// index would file them in different buckets and append a duplicate where the
// linear scan deduped. Two consequences of that contract are non-obvious:
// -0.0 is normalized to +0.0 (Equal compares doubles with ==), and maps are
// combined order-independently (Map.Equal matches by key, not by position).
func hashValue(h uint64, v pcommon.Value) uint64 {
	h = mixU64(h, uint64(v.Type()))

	switch v.Type() {
	case pcommon.ValueTypeStr:
		return mixString(h, v.Str())
	case pcommon.ValueTypeInt:
		return mixU64(h, uint64(v.Int()))
	case pcommon.ValueTypeDouble:
		bits := math.Float64bits(v.Double())
		if bits == negZeroBits {
			bits = 0
		}
		return mixU64(h, bits)
	case pcommon.ValueTypeBool:
		if v.Bool() {
			return mixU64(h, 1)
		}
		return mixU64(h, 0)
	case pcommon.ValueTypeBytes:
		return mixBytes(h, v.Bytes())
	case pcommon.ValueTypeSlice:
		s := v.Slice()
		h = mixU64(h, uint64(s.Len()))
		for i := 0; i < s.Len(); i++ {
			h = hashValue(h, s.At(i))
		}
		return h
	case pcommon.ValueTypeMap:
		m := v.Map()
		var acc uint64
		for k, mv := range m.All() {
			acc += hashValue(mixString(fnvOffset64, k), mv)
		}
		return mixU64(mixU64(h, uint64(m.Len())), acc)
	case pcommon.ValueTypeEmpty:
	}

	return h
}

func hashFunction(fn Function) uint64 {
	h := fnvOffset64
	h = mixU64(h, uint64(uint32(fn.NameStrindex())))
	h = mixU64(h, uint64(uint32(fn.SystemNameStrindex())))
	h = mixU64(h, uint64(uint32(fn.FilenameStrindex())))
	h = mixU64(h, uint64(fn.StartLine()))
	return h
}

func hashMapping(ma Mapping) uint64 {
	h := fnvOffset64
	h = mixU64(h, ma.MemoryStart())
	h = mixU64(h, ma.MemoryLimit())
	h = mixU64(h, ma.FileOffset())
	h = mixU64(h, uint64(uint32(ma.FilenameStrindex())))
	ai := ma.AttributeIndices()
	for i := 0; i < ai.Len(); i++ {
		h = mixU64(h, uint64(uint32(ai.At(i))))
	}
	return h
}

func hashLocation(loc Location) uint64 {
	h := fnvOffset64
	h = mixU64(h, uint64(uint32(loc.MappingIndex())))
	h = mixU64(h, loc.Address())
	ai := loc.AttributeIndices()
	for i := 0; i < ai.Len(); i++ {
		h = mixU64(h, uint64(uint32(ai.At(i))))
	}
	lines := loc.Lines()
	for i := 0; i < lines.Len(); i++ {
		ln := lines.At(i)
		h = mixU64(h, uint64(uint32(ln.FunctionIndex())))
		h = mixU64(h, uint64(ln.Line()))
		h = mixU64(h, uint64(ln.Column()))
	}
	return h
}

func hashStack(st Stack) uint64 {
	h := fnvOffset64
	li := st.LocationIndices()
	for i := 0; i < li.Len(); i++ {
		h = mixU64(h, uint64(uint32(li.At(i))))
	}
	return h
}

func hashAttribute(a KeyValueAndUnit) uint64 {
	return hashAttributeFields(a.KeyStrindex(), a.UnitStrindex(), a.Value())
}

func hashAttributeFields(key, unit int32, value pcommon.Value) uint64 {
	h := fnvOffset64
	h = mixU64(h, uint64(uint32(key)))
	h = mixU64(h, uint64(uint32(unit)))
	h = hashValue(h, value)
	return h
}

func hashLink(li Link) uint64 {
	h := fnvOffset64
	tid := li.TraceID()
	for _, b := range tid {
		h = mixU64(h, uint64(b))
	}
	sid := li.SpanID()
	for _, b := range sid {
		h = mixU64(h, uint64(b))
	}
	return h
}

// mergeIndex accelerates dictionary dedup during a single MergeTo via per-table hash buckets; the table's Equal stays the dedup arbiter.
type mergeIndex struct {
	// maxTableLen is the point at which a dictionary table can no longer grow
	// without overflowing an int32 index. Carrying it per index rather than as
	// a package-level constant lets tests exercise the overflow guards without
	// allocating billions of entries.
	maxTableLen int

	strings    map[string]int32
	functions  map[uint64][]int32
	mappings   map[uint64][]int32
	locations  map[uint64][]int32
	stacks     map[uint64][]int32
	attributes map[uint64][]int32
	links      map[uint64][]int32

	// Source index to destination index for the merge in progress, so each
	// referenced source entry is translated and copied at most once. Sized on
	// first use; -1 marks an entry not translated yet.
	// appendOnly skips deduplication: each translated entry is appended.
	// Only valid when translating from a single source dictionary into a new
	// one, where the index memos already copy each source entry once.
	appendOnly bool

	attributeIdx indexMemo
	functionIdx  indexMemo
	mappingIdx   indexMemo
	locationIdx  indexMemo
	stackIdx     indexMemo

	// Scratch values an entry is copied into and translated in before being
	// deduplicated into the destination. One per type is enough because a
	// translation never recurses into another entry of the same type.
	functionScratch *Function
	mappingScratch  *Mapping
	locationScratch *Location
	stackScratch    *Stack
}

func newMergeIndex(dst ProfilesDictionary) *mergeIndex {
	mi := &mergeIndex{
		maxTableLen: math.MaxInt32,

		strings:    make(map[string]int32, dst.StringTable().Len()),
		functions:  make(map[uint64][]int32, dst.FunctionTable().Len()),
		mappings:   make(map[uint64][]int32, dst.MappingTable().Len()),
		locations:  make(map[uint64][]int32, dst.LocationTable().Len()),
		stacks:     make(map[uint64][]int32, dst.StackTable().Len()),
		attributes: make(map[uint64][]int32, dst.AttributeTable().Len()),
		links:      make(map[uint64][]int32, dst.LinkTable().Len()),
	}

	st := dst.StringTable()
	for i := 0; i < st.Len(); i++ {
		s := st.At(i)
		if _, ok := mi.strings[s]; !ok {
			mi.strings[s] = int32(i)
		}
	}
	ft := dst.FunctionTable()
	for i := 0; i < ft.Len(); i++ {
		h := hashFunction(ft.At(i))
		mi.functions[h] = append(mi.functions[h], int32(i))
	}
	mt := dst.MappingTable()
	for i := 0; i < mt.Len(); i++ {
		h := hashMapping(mt.At(i))
		mi.mappings[h] = append(mi.mappings[h], int32(i))
	}
	lt := dst.LocationTable()
	for i := 0; i < lt.Len(); i++ {
		h := hashLocation(lt.At(i))
		mi.locations[h] = append(mi.locations[h], int32(i))
	}
	stkt := dst.StackTable()
	for i := 0; i < stkt.Len(); i++ {
		h := hashStack(stkt.At(i))
		mi.stacks[h] = append(mi.stacks[h], int32(i))
	}
	at := dst.AttributeTable()
	for i := 0; i < at.Len(); i++ {
		h := hashAttribute(at.At(i))
		mi.attributes[h] = append(mi.attributes[h], int32(i))
	}
	lkt := dst.LinkTable()
	for i := 0; i < lkt.Len(); i++ {
		h := hashLink(lkt.At(i))
		mi.links[h] = append(mi.links[h], int32(i))
	}

	return mi
}

func (mi *mergeIndex) setString(table pcommon.StringSlice, val string) (int32, error) {
	if idx, ok := mi.strings[val]; ok {
		return idx, nil
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyStringTableEntries
	}
	table.Append(val)
	idx := int32(table.Len() - 1)
	mi.strings[val] = idx
	return idx, nil
}

func (mi *mergeIndex) setFunction(table FunctionSlice, fn Function) (int32, error) {
	var h uint64
	if !mi.appendOnly {
		h = hashFunction(fn)
	}
	for _, idx := range mi.functions[h] {
		if table.At(int(idx)).Equal(fn) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyFunctionTableEntries
	}
	fn.CopyTo(table.AppendEmpty())
	idx := int32(table.Len() - 1)
	if !mi.appendOnly {
		mi.functions[h] = append(mi.functions[h], idx)
	}
	return idx, nil
}

func (mi *mergeIndex) setMapping(table MappingSlice, ma Mapping) (int32, error) {
	var h uint64
	if !mi.appendOnly {
		h = hashMapping(ma)
	}
	for _, idx := range mi.mappings[h] {
		if table.At(int(idx)).Equal(ma) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyMappingTableEntries
	}
	ma.CopyTo(table.AppendEmpty())
	idx := int32(table.Len() - 1)
	if !mi.appendOnly {
		mi.mappings[h] = append(mi.mappings[h], idx)
	}
	return idx, nil
}

func (mi *mergeIndex) setLocation(table LocationSlice, loc Location) (int32, error) {
	var h uint64
	if !mi.appendOnly {
		h = hashLocation(loc)
	}
	for _, idx := range mi.locations[h] {
		if table.At(int(idx)).Equal(loc) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyLocationTableEntries
	}
	loc.CopyTo(table.AppendEmpty())
	idx := int32(table.Len() - 1)
	if !mi.appendOnly {
		mi.locations[h] = append(mi.locations[h], idx)
	}
	return idx, nil
}

func (mi *mergeIndex) setStack(table StackSlice, st Stack) (int32, error) {
	var h uint64
	if !mi.appendOnly {
		h = hashStack(st)
	}
	for _, idx := range mi.stacks[h] {
		if table.At(int(idx)).Equal(st) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyStackTableEntries
	}
	st.CopyTo(table.AppendEmpty())
	idx := int32(table.Len() - 1)
	if !mi.appendOnly {
		mi.stacks[h] = append(mi.stacks[h], idx)
	}
	return idx, nil
}

func (mi *mergeIndex) setAttribute(table KeyValueAndUnitSlice, attr KeyValueAndUnit) (int32, error) {
	return mi.setAttributeFields(table, attr.KeyStrindex(), attr.UnitStrindex(), attr.Value())
}

// setAttributeFields is setAttribute for an attribute given by its fields, so
// callers can translate the key and unit without copying the value first.
func (mi *mergeIndex) setAttributeFields(table KeyValueAndUnitSlice, key, unit int32, value pcommon.Value) (int32, error) {
	var h uint64
	if !mi.appendOnly {
		h = hashAttributeFields(key, unit, value)
	}
	for _, idx := range mi.attributes[h] {
		e := table.At(int(idx))
		if e.KeyStrindex() == key && e.UnitStrindex() == unit && e.Value().Equal(value) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyAttributeTableEntries
	}
	e := table.AppendEmpty()
	e.SetKeyStrindex(key)
	e.SetUnitStrindex(unit)
	value.CopyTo(e.Value())
	idx := int32(table.Len() - 1)
	if !mi.appendOnly {
		mi.attributes[h] = append(mi.attributes[h], idx)
	}
	return idx, nil
}

func (mi *mergeIndex) setLink(table LinkSlice, li Link) (int32, error) {
	h := hashLink(li)
	for _, idx := range mi.links[h] {
		if table.At(int(idx)).Equal(li) {
			return idx, nil
		}
	}
	if table.Len() >= mi.maxTableLen {
		return 0, errTooManyLinkTableEntries
	}
	li.CopyTo(table.AppendEmpty())
	idx := int32(table.Len() - 1)
	mi.links[h] = append(mi.links[h], idx)
	return idx, nil
}

// newSplitIndex returns a mergeIndex for translating part of a single source
// dictionary into a new dst holding only the index 0 empty entries: index
// memos are sparse, and entries are appended without deduplication.
func newSplitIndex(dst ProfilesDictionary) *mergeIndex {
	mi := newMergeIndex(dst)
	mi.appendOnly = true
	for _, memo := range []*indexMemo{&mi.attributeIdx, &mi.functionIdx, &mi.mappingIdx, &mi.locationIdx, &mi.stackIdx} {
		memo.sparse = true
	}
	return mi
}

// indexMemo maps the source indices of one dictionary table to destination
// indices. A dense memo is a slice sized to the source table, suited to merging
// most of a dictionary; a sparse memo is a map, suited to translating a few
// entries of a large dictionary, such as when splitting it.
type indexMemo struct {
	sparse bool
	dense  []int32
	m      map[int32]int32
}

// get returns the destination index of source index i of a table of length n.
func (memo *indexMemo) get(n int, i int32) (int32, bool) {
	if memo.sparse {
		v, ok := memo.m[i]
		return v, ok
	}
	if len(memo.dense) != n {
		memo.dense = make([]int32, n)
		for j := range memo.dense {
			memo.dense[j] = -1
		}
	}
	v := memo.dense[i]
	return v, v >= 0
}

// set records the destination index of source index i. get must have been
// called for i first.
func (memo *indexMemo) set(i, v int32) {
	if memo.sparse {
		if memo.m == nil {
			memo.m = make(map[int32]int32)
		}
		memo.m[i] = v
		return
	}
	memo.dense[i] = v
}

// attribute returns the destination index of source attribute i. On first use
// it copies the attribute, with its own references translated, into dst.
func (mi *mergeIndex) attribute(src, dst ProfilesDictionary, i int32) (int32, error) {
	if mi.appendOnly && i == 0 {
		return 0, nil
	}
	if idx, ok := mi.attributeIdx.get(src.AttributeTable().Len(), i); ok {
		return idx, nil
	}
	attr := src.AttributeTable().At(int(i))
	key, unit, err := attr.translateStrings(src, dst, mi)
	if err != nil {
		return 0, err
	}
	idx, err := mi.setAttributeFields(dst.AttributeTable(), key, unit, attr.Value())
	if err != nil {
		return 0, err
	}
	mi.attributeIdx.set(i, idx)
	return idx, nil
}

// function returns the destination index of source function i. On first use
// it copies the function, with its own references translated, into dst.
func (mi *mergeIndex) function(src, dst ProfilesDictionary, i int32) (int32, error) {
	if mi.appendOnly && i == 0 {
		return 0, nil
	}
	if idx, ok := mi.functionIdx.get(src.FunctionTable().Len(), i); ok {
		return idx, nil
	}
	if mi.functionScratch == nil {
		v := NewFunction()
		mi.functionScratch = &v
	}
	fn := *mi.functionScratch
	src.FunctionTable().At(int(i)).CopyTo(fn)
	if err := fn.switchDictionary(src, dst, mi); err != nil {
		return 0, err
	}
	idx, err := mi.setFunction(dst.FunctionTable(), fn)
	if err != nil {
		return 0, err
	}
	mi.functionIdx.set(i, idx)
	return idx, nil
}

// mapping returns the destination index of source mapping i. On first use it
// copies the mapping, with its own references translated, into dst.
func (mi *mergeIndex) mapping(src, dst ProfilesDictionary, i int32) (int32, error) {
	if mi.appendOnly && i == 0 {
		return 0, nil
	}
	if idx, ok := mi.mappingIdx.get(src.MappingTable().Len(), i); ok {
		return idx, nil
	}
	if mi.mappingScratch == nil {
		v := NewMapping()
		mi.mappingScratch = &v
	}
	ma := *mi.mappingScratch
	src.MappingTable().At(int(i)).CopyTo(ma)
	if err := ma.switchDictionary(src, dst, mi); err != nil {
		return 0, err
	}
	idx, err := mi.setMapping(dst.MappingTable(), ma)
	if err != nil {
		return 0, err
	}
	mi.mappingIdx.set(i, idx)
	return idx, nil
}

// location returns the destination index of source location i. On first use
// it copies the location, with its own references translated, into dst.
func (mi *mergeIndex) location(src, dst ProfilesDictionary, i int32) (int32, error) {
	if mi.appendOnly && i == 0 {
		return 0, nil
	}
	if idx, ok := mi.locationIdx.get(src.LocationTable().Len(), i); ok {
		return idx, nil
	}
	if mi.locationScratch == nil {
		v := NewLocation()
		mi.locationScratch = &v
	}
	loc := *mi.locationScratch
	src.LocationTable().At(int(i)).CopyTo(loc)
	if err := loc.switchDictionary(src, dst, mi); err != nil {
		return 0, err
	}
	idx, err := mi.setLocation(dst.LocationTable(), loc)
	if err != nil {
		return 0, err
	}
	mi.locationIdx.set(i, idx)
	return idx, nil
}

// stack returns the destination index of source stack i. On first use it
// copies the stack, with its own references translated, into dst.
func (mi *mergeIndex) stack(src, dst ProfilesDictionary, i int32) (int32, error) {
	if mi.appendOnly && i == 0 {
		return 0, nil
	}
	if idx, ok := mi.stackIdx.get(src.StackTable().Len(), i); ok {
		return idx, nil
	}
	if mi.stackScratch == nil {
		v := NewStack()
		mi.stackScratch = &v
	}
	st := *mi.stackScratch
	src.StackTable().At(int(i)).CopyTo(st)
	if err := st.switchDictionary(src, dst, mi); err != nil {
		return 0, err
	}
	idx, err := mi.setStack(dst.StackTable(), st)
	if err != nil {
		return 0, err
	}
	mi.stackIdx.set(i, idx)
	return idx, nil
}
