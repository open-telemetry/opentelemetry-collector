// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"go.opencensus.io/trace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
)

// canonicalCodes maps (*trace.SpanData).Status.Code to their description. See:
// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto.
var canonicalCodes = [...]string{
	"ok",
	"cancelled",
	"unknown",
	"invalid_argument",
	"deadline_exceeded",
	"not_found",
	"already_exists",
	"permission_denied",
	"resource_exhausted",
	"failed_precondition",
	"aborted",
	"out_of_range",
	"unimplemented",
	"internal",
	"unavailable",
	"data_loss",
	"unauthenticated",
}

func canonicalCodeString(code int32) string {
	if code < 0 || int(code) >= len(canonicalCodes) {
		return "error code " + strconv.FormatInt(int64(code), 10)
	}
	return canonicalCodes[code]
}

// convertSpan takes an OpenCensus span and returns a Datadog span.
func (e *traceExporter) convertSpan(s *trace.SpanData) *ddSpan {
	startNano := s.StartTime.UnixNano()
	span := &ddSpan{
		TraceID:  binary.BigEndian.Uint64(s.SpanContext.TraceID[8:]),
		SpanID:   binary.BigEndian.Uint64(s.SpanContext.SpanID[:]),
		Name:     s.Name,
		Resource: s.Name,
		Service:  e.opts.Service,
		Start:    startNano,
		Duration: s.EndTime.UnixNano() - startNano,
		Metrics:  map[string]float64{samplingPriorityKey: ext.PriorityAutoKeep},
		Meta:     map[string]string{},
	}
	if s.ParentSpanID != (trace.SpanID{}) {
		span.ParentID = binary.BigEndian.Uint64(s.ParentSpanID[:])
	}
	switch s.SpanKind {
	case trace.SpanKindClient:
		span.Type = "client"
	case trace.SpanKindServer:
		span.Type = "server"
	}
	statusKey := statusDescriptionKey
	if code := s.Status.Code; code != 0 {
		statusKey = ext.ErrorMsg
		span.Error = 1
		span.Meta[ext.ErrorType] = canonicalCodeString(s.Status.Code)
	}
	if msg := s.Status.Message; msg != "" {
		span.Meta[statusKey] = msg
	}
	for key, val := range s.Attributes {
		setTag(span, key, val)
	}
	return span
}

const (
	samplingPriorityKey  = "_sampling_priority_v1"
	statusDescriptionKey = "opencensus.status_description"
)

func setTag(s *ddSpan, key string, val interface{}) {
	if key == ext.Error {
		setError(s, val)
		return
	}
	switch v := val.(type) {
	case string:
		setStringTag(s, key, v)
		return
	case bool:
		if v {
			s.Meta[key] = "true"
		} else {
			s.Meta[key] = "false"
		}
	case int64:
		if key == ext.SamplingPriority {
			s.Metrics[samplingPriorityKey] = float64(v)
		} else {
			s.Metrics[key] = float64(v)
		}
	default:
		// should never happen according to docs, nevertheless
		// we should account for this to avoid exceptions
		s.Meta[key] = fmt.Sprintf("%v", v)
	}
}

func setStringTag(s *ddSpan, key, v string) {
	switch key {
	case ext.ServiceName:
		s.Service = v
	case ext.ResourceName:
		s.Resource = v
	case ext.SpanType:
		s.Type = v
	default:
		s.Meta[key] = v
	}
}

func setError(s *ddSpan, val interface{}) {
	switch v := val.(type) {
	case string:
		s.Error = 1
		s.Meta[ext.ErrorMsg] = v
	case bool:
		if v {
			s.Error = 1
		} else {
			s.Error = 0
		}
	case int64:
		if v > 0 {
			s.Error = 1
		} else {
			s.Error = 0
		}
	case nil:
		s.Error = 0
	default:
		s.Error = 1
	}
}
