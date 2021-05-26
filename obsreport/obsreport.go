// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package obsreport

import (
	"context"
	"strings"

	"go.opencensus.io/trace"

	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

var (
	okStatus = trace.Status{Code: trace.StatusCodeOK}
)

// setParentLink tries to retrieve a span from parentCtx and if one exists
// sets its SpanID, TraceID as a link to the given child Span.
// It returns true only if it retrieved a parent span from the context.
//
// This is typically used when the parentCtx may already have a trace and is
// long lived (eg.: an gRPC stream, or TCP connection) and one desires distinct
// traces for individual operations under the long lived trace associated to
// the parentCtx. This function is a helper that encapsulates the work of
// linking the short lived trace/span to the longer one.
func setParentLink(parentCtx context.Context, childSpan *trace.Span) bool {
	parentSpanFromRPC := trace.FromContext(parentCtx)
	if parentSpanFromRPC == nil {
		return false
	}

	psc := parentSpanFromRPC.SpanContext()
	childSpan.AddLink(trace.Link{
		SpanID:  psc.SpanID,
		TraceID: psc.TraceID,
		Type:    trace.LinkTypeParent,
	})
	return true
}

func buildComponentPrefix(componentPrefix, configType string) string {
	if !strings.HasSuffix(componentPrefix, obsmetrics.NameSep) {
		componentPrefix += obsmetrics.NameSep
	}
	if configType == "" {
		return componentPrefix
	}
	return componentPrefix + configType + obsmetrics.NameSep
}

func errToStatus(err error) trace.Status {
	if err != nil {
		return trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()}
	}
	return okStatus
}
