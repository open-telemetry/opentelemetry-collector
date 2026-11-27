// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/pdata/internal"

import "unicode/utf8"

// ValidateUTF8ExportTraceServiceRequest returns false when any string in req contains invalid UTF-8.
func ValidateUTF8ExportTraceServiceRequest(req *ExportTraceServiceRequest) bool {
	for _, rs := range req.ResourceSpans {
		if !validUTF8Resource(&rs.Resource) || !utf8.ValidString(rs.SchemaUrl) ||
			!validUTF8ScopeSpansSlice(rs.ScopeSpans) || !validUTF8ScopeSpansSlice(rs.DeprecatedScopeSpans) {
			return false
		}
	}
	return true
}

func validUTF8ScopeSpansSlice(scopes []*ScopeSpans) bool {
	for _, ss := range scopes {
		if !validUTF8InstrumentationScope(&ss.Scope) || !utf8.ValidString(ss.SchemaUrl) {
			return false
		}
		for _, span := range ss.Spans {
			if !validUTF8Span(span) {
				return false
			}
		}
	}
	return true
}

func validUTF8Span(span *Span) bool {
	if !utf8.ValidString(span.TraceState) || !utf8.ValidString(span.Name) ||
		!utf8.ValidString(span.Status.Message) || !validUTF8KeyValueSlice(span.Attributes) {
		return false
	}
	for _, e := range span.Events {
		if !utf8.ValidString(e.Name) || !validUTF8KeyValueSlice(e.Attributes) {
			return false
		}
	}
	for _, l := range span.Links {
		if !utf8.ValidString(l.TraceState) || !validUTF8KeyValueSlice(l.Attributes) {
			return false
		}
	}
	return true
}

// RejectInvalidUTF8ExportTraceServiceRequest removes spans containing invalid UTF-8 from req
// and returns the number removed.
func RejectInvalidUTF8ExportTraceServiceRequest(req *ExportTraceServiceRequest) int {
	rejected := 0
	req.ResourceSpans = removeIf(req.ResourceSpans, DeleteResourceSpans, func(rs *ResourceSpans) bool {
		if !validUTF8Resource(&rs.Resource) || !utf8.ValidString(rs.SchemaUrl) {
			for _, ss := range rs.ScopeSpans {
				rejected += len(ss.Spans)
			}
			return true
		}
		if len(rs.ScopeSpans) == 0 {
			return false
		}
		rs.ScopeSpans = removeIf(rs.ScopeSpans, DeleteScopeSpans, func(ss *ScopeSpans) bool {
			if !validUTF8InstrumentationScope(&ss.Scope) || !utf8.ValidString(ss.SchemaUrl) {
				rejected += len(ss.Spans)
				return true
			}
			if len(ss.Spans) == 0 {
				return false
			}
			ss.Spans = removeIf(ss.Spans, DeleteSpan, func(span *Span) bool {
				invalid := !validUTF8Span(span)
				if invalid {
					rejected++
				}
				return invalid
			})
			return len(ss.Spans) == 0
		})
		return len(rs.ScopeSpans) == 0
	})
	return rejected
}
