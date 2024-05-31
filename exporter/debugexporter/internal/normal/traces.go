// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package normal

import (
	"bytes"
	"fmt"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

type normalTracesMarshaler struct{}

// Ensure normalTracesMarshaller implements interface ptrace.Marshaler
var _ ptrace.Marshaler = normalTracesMarshaler{}

// NewNormalTracesMarshaler returns a ptrace.Marshaler for normal verbosity. It writes one line of text per log record
func NewNormalTracesMarshaler() ptrace.Marshaler {
	return normalTracesMarshaler{}
}

func (normalTracesMarshaler) MarshalTraces(md ptrace.Traces) ([]byte, error) {
	var buffer bytes.Buffer
	for i := 0; i < md.ResourceSpans().Len(); i++ {
		resourceTraces := md.ResourceSpans().At(i)
		for j := 0; j < resourceTraces.ScopeSpans().Len(); j++ {
			scopeTraces := resourceTraces.ScopeSpans().At(j)
			for k := 0; k < scopeTraces.Spans().Len(); k++ {
				span := scopeTraces.Spans().At(k)
				spanString := fmt.Sprintf("%s %s %s", span.Name(), span.TraceID().String(), span.SpanID().String())
				buffer.WriteString(spanString)
				buffer.WriteString("\n")
			}
		}
	}
	return buffer.Bytes(), nil
}
