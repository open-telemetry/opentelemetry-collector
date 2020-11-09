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

package v1

import (
	context "context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// The aliases in this file are necessary to fix the bug:
// https://github.com/open-telemetry/opentelemetry-collector/issues/1968

// patternTraceServiceExport0Alias is an alias for the incorrect pattern
// pattern_TraceService_Export_0 defined in trace_service.pb.gw.go.
//
// The path in the pattern_TraceService_Export_0 pattern is incorrect because it is
// composed from the historical name of the package v1.trace used in the Protobuf
// declarations in trace_service.proto file and results in the path of /v1/trace.
//
// This is incorrect since the OTLP spec requires the default path to be /v1/traces,
// see https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/protocol/otlp.md#request.
//
// We set the correct path in this alias.
var patternTraceServiceExport0Alias = runtime.MustPattern(
	runtime.NewPattern(
		1,
		[]int{2, 0, 2, 1},
		[]string{"v1", "traces"}, // Patch the path to be /v1/traces.
		"",
		runtime.AssumeColonVerbOpt(true)),
)

// RegisterTraceServiceHandlerServerAlias registers the http handlers for service TraceService to "mux".
// UnaryRPC     :call TraceServiceServer directly.
// StreamingRPC :currently unsupported pending https://github.com/grpc/grpc-go/issues/906.
//
// RegisterTraceServiceHandlerServerAlias is the alias version of
// RegisterTraceServiceHandlerServer, and uses patternTraceServiceExport0Alias
// instead of pattern_TraceService_Export_0.
func RegisterTraceServiceHandlerServerAlias(ctx context.Context, mux *runtime.ServeMux, server TraceServiceServer) error {

	// pattern_TraceService_Export_0 is replaced by patternTraceServiceExport0Alias
	// in the following line. This is the only change in this func compared to
	// RegisterTraceServiceHandlerServer.
	mux.Handle("POST", patternTraceServiceExport0Alias, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateIncomingContext(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err := local_request_TraceService_Export_0(rctx, inboundMarshaler, server, req, pathParams)
		ctx = runtime.NewServerMetadataContext(ctx, md)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}

		forward_TraceService_Export_0(ctx, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}
