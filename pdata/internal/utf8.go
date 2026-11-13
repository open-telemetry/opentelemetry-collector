// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import (
	"strings"
	"unicode/utf8"
)

// utf8Replacement is the Unicode replacement character used for invalid UTF-8 byte sequences.
const utf8Replacement = "�"

// SanitizeUTF8String returns s unchanged when it is valid UTF-8. Otherwise it returns a copy of s
// where every run of invalid UTF-8 bytes is replaced with the Unicode replacement character U+FFFD.
func SanitizeUTF8String(s string) string {
	// utf8.ValidString has an ASCII fast path and does not allocate, so valid strings (the common case)
	// cost a single scan. strings.ToValidUTF8 only allocates when there is something to replace.
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, utf8Replacement)
}

// SanitizeUTF8StringSlice sanitizes every string in ss in place.
func SanitizeUTF8StringSlice(ss []string) {
	for i := range ss {
		ss[i] = SanitizeUTF8String(ss[i])
	}
}

// SanitizeUTF8AnyValue sanitizes the string value and all nested string values and keys of v in place.
// Bytes values and string table indices are left untouched.
func SanitizeUTF8AnyValue(v *AnyValue) {
	switch ov := v.Value.(type) {
	case *AnyValue_StringValue:
		ov.StringValue = SanitizeUTF8String(ov.StringValue)
	case *AnyValue_ArrayValue:
		if ov.ArrayValue != nil {
			SanitizeUTF8AnyValueSlice(ov.ArrayValue.Values)
		}
	case *AnyValue_KvlistValue:
		if ov.KvlistValue != nil {
			SanitizeUTF8KeyValueSlice(ov.KvlistValue.Values)
		}
	}
}

// SanitizeUTF8AnyValueSlice sanitizes every value in vs in place.
func SanitizeUTF8AnyValueSlice(vs []AnyValue) {
	for i := range vs {
		SanitizeUTF8AnyValue(&vs[i])
	}
}

// SanitizeUTF8KeyValueSlice sanitizes every key and value in kvs in place.
func SanitizeUTF8KeyValueSlice(kvs []KeyValue) {
	for i := range kvs {
		kvs[i].Key = SanitizeUTF8String(kvs[i].Key)
		SanitizeUTF8AnyValue(&kvs[i].Value)
	}
}

// SanitizeUTF8Resource sanitizes the attributes and entity references of r in place.
func SanitizeUTF8Resource(r *Resource) {
	SanitizeUTF8KeyValueSlice(r.Attributes)
	for _, er := range r.EntityRefs {
		er.SchemaUrl = SanitizeUTF8String(er.SchemaUrl)
		er.Type = SanitizeUTF8String(er.Type)
		SanitizeUTF8StringSlice(er.IdKeys)
		SanitizeUTF8StringSlice(er.DescriptionKeys)
	}
}

// SanitizeUTF8InstrumentationScope sanitizes the name, version and attributes of s in place.
func SanitizeUTF8InstrumentationScope(s *InstrumentationScope) {
	s.Name = SanitizeUTF8String(s.Name)
	s.Version = SanitizeUTF8String(s.Version)
	SanitizeUTF8KeyValueSlice(s.Attributes)
}
