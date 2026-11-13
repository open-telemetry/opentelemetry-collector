// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statusutil // import "go.opentelemetry.io/collector/internal/statusutil"

import (
	"sync"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	// MaxDetailDepth is the number of levels of google.protobuf.Any messages, nested inside
	// one another, that BoundDetails keeps. A detail nested deeper is reduced to its type URL.
	MaxDetailDepth = 8

	// MaxDetailBytes is the size of the largest detail that BoundDetails decodes and keeps.
	// A larger detail is reduced to its type URL.
	MaxDetailBytes = 64 * 1024

	// MaxDecodedDetails is the number of google.protobuf.Any messages, nested ones included,
	// that BoundDetails decodes for a single status. The others are reduced to their type URL.
	MaxDecodedDetails = 32

	anyFullName protoreflect.FullName = "google.protobuf.Any"
)

// BoundDetails returns s with the details bounded in depth, size and number, so that the status can be
// safely expanded by an encoder such as protojson or prototext. Such encoders resolve every
// google.protobuf.Any recursively and keep a copy of the nested bytes at each level, so the memory
// they need grows quadratically with the nesting depth of a crafted status.
//
// Use it for a status that may come from an untrusted peer, such as the backend of an exporter. A detail
// that is within the limits is kept as it is. One that exceeds them is reduced to its type URL, so the
// type of the detail is still reported. s is not modified; if nothing exceeds the limits s is returned.
func BoundDetails(s *status.Status) *status.Status {
	p := s.Proto()
	if len(p.GetDetails()) == 0 {
		return s
	}
	b := &detailsBounder{budget: MaxDecodedDetails}
	changed := false
	for i, d := range p.Details {
		if bounded := b.boundAny(d, 1); bounded != d {
			p.Details[i] = bounded
			changed = true
		}
	}
	if !changed {
		return s
	}
	return status.FromProto(p)
}

type detailsBounder struct {
	// budget is the number of messages that can still be decoded.
	budget int
}

// boundAny returns a, or a replacement for it when a, or something nested in it, exceeds the limits.
// depth is the nesting level of a, starting at 1.
func (b *detailsBounder) boundAny(a *anypb.Any, depth int) *anypb.Any {
	if depth > MaxDetailDepth || b.budget <= 0 || len(a.GetValue()) > MaxDetailBytes {
		return &anypb.Any{TypeUrl: a.GetTypeUrl()}
	}
	mt, err := protoregistry.GlobalTypes.FindMessageByURL(a.GetTypeUrl())
	if err != nil || !containsAny(mt.Descriptor()) {
		// An unknown type is not expanded by encoders. A type that cannot hold a google.protobuf.Any
		// has nothing nested to bound, so it is left undecoded. Its size is within MaxDetailBytes.
		return a
	}
	b.budget--
	m, err := a.UnmarshalNew()
	if err != nil {
		// Malformed value.
		return a
	}
	if !b.boundMessage(m.ProtoReflect(), depth) {
		return a
	}
	bounded, err := anypb.New(m)
	if err != nil {
		return &anypb.Any{TypeUrl: a.GetTypeUrl()}
	}
	return bounded
}

// boundMessage bounds every google.protobuf.Any found in m, in place. depth is the nesting
// level of the Any that holds m. It reports whether m was changed.
func (b *detailsBounder) boundMessage(m protoreflect.Message, depth int) bool {
	type update struct {
		fd protoreflect.FieldDescriptor
		v  protoreflect.Value
	}
	var updates []update
	changed := false
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Message() != nil:
			l := v.List()
			for i := 0; i < l.Len(); i++ {
				if nv, ok := b.boundValue(l.Get(i), depth); ok {
					l.Set(i, nv)
					changed = true
				}
			}
		case fd.IsMap() && fd.MapValue().Message() != nil:
			mp := v.Map()
			var keys []protoreflect.MapKey
			var values []protoreflect.Value
			mp.Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
				if nv, ok := b.boundValue(mv, depth); ok {
					keys = append(keys, k)
					values = append(values, nv)
				}
				return true
			})
			for i, k := range keys {
				mp.Set(k, values[i])
				changed = true
			}
		case !fd.IsList() && !fd.IsMap() && fd.Message() != nil:
			if nv, ok := b.boundValue(v, depth); ok {
				updates = append(updates, update{fd, nv})
				changed = true
			}
		}
		return true
	})
	for _, u := range updates {
		m.Set(u.fd, u.v)
	}
	return changed
}

// boundValue bounds a message value found at nesting level depth. It returns the
// value to store in place of v and whether anything changed.
func (b *detailsBounder) boundValue(v protoreflect.Value, depth int) (protoreflect.Value, bool) {
	msg := v.Message()
	if msg.Descriptor().FullName() != anyFullName {
		return v, b.boundMessage(msg, depth)
	}
	a, ok := msg.Interface().(*anypb.Any)
	if !ok {
		return v, false
	}
	bounded := b.boundAny(a, depth+1)
	if bounded == a {
		return v, false
	}
	return protoreflect.ValueOfMessage(bounded.ProtoReflect()), true
}

// anyCache memoizes containsAny by message type.
var anyCache sync.Map // protoreflect.FullName -> bool

// containsAny reports whether a message of type md can hold a google.protobuf.Any, however deep.
func containsAny(md protoreflect.MessageDescriptor) bool {
	if v, ok := anyCache.Load(md.FullName()); ok {
		return v.(bool)
	}
	found := searchAny(md, map[protoreflect.FullName]bool{})
	anyCache.Store(md.FullName(), found)
	return found
}

func searchAny(md protoreflect.MessageDescriptor, visited map[protoreflect.FullName]bool) bool {
	if md.FullName() == anyFullName {
		return true
	}
	if visited[md.FullName()] {
		return false
	}
	visited[md.FullName()] = true
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		if fm := fields.Get(i).Message(); fm != nil && searchAny(fm, visited) {
			return true
		}
	}
	return false
}
