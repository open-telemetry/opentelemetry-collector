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

package otlpreceiver

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
)

var jsonMarshaler = &jsonpb.Marshaler{}

func handleTraces(
	resp http.ResponseWriter,
	req *http.Request,
	contentType string,
	tracesReceiver *trace.Receiver,
	tracesUnmarshaler pdata.TracesUnmarshaler) {
	body, ok := readAndCloseBody(resp, req, contentType)
	if !ok {
		return
	}

	td, err := tracesUnmarshaler.UnmarshalTraces(body)
	if err != nil {
		writeError(resp, contentType, err, http.StatusBadRequest)
		return
	}

	_, err = tracesReceiver.Export(req.Context(), td)
	if err != nil {
		writeError(resp, contentType, err, http.StatusInternalServerError)
		return
	}

	// TODO: Pass response from grpc handler when otlpgrpc returns concrete type.
	writeResponse(resp, contentType, http.StatusOK, &types.Empty{})
}

func handleMetrics(
	resp http.ResponseWriter,
	req *http.Request,
	contentType string,
	metricsReceiver *metrics.Receiver,
	metricsUnmarshaler pdata.MetricsUnmarshaler) {
	body, ok := readAndCloseBody(resp, req, contentType)
	if !ok {
		return
	}

	md, err := metricsUnmarshaler.UnmarshalMetrics(body)
	if err != nil {
		writeError(resp, contentType, err, http.StatusBadRequest)
		return
	}

	_, err = metricsReceiver.Export(req.Context(), md)
	if err != nil {
		writeError(resp, contentType, err, http.StatusInternalServerError)
		return
	}

	// TODO: Pass response from grpc handler when otlpgrpc returns concrete type.
	writeResponse(resp, contentType, http.StatusOK, &types.Empty{})
}

func handleLogs(
	resp http.ResponseWriter,
	req *http.Request,
	contentType string,
	logsReceiver *logs.Receiver,
	logsUnmarshaler pdata.LogsUnmarshaler) {
	body, ok := readAndCloseBody(resp, req, contentType)
	if !ok {
		return
	}

	ld, err := logsUnmarshaler.UnmarshalLogs(body)
	if err != nil {
		writeError(resp, contentType, err, http.StatusBadRequest)
		return
	}

	_, err = logsReceiver.Export(req.Context(), ld)
	if err != nil {
		writeError(resp, contentType, err, http.StatusInternalServerError)
		return
	}

	// TODO: Pass response from grpc handler when otlpgrpc returns concrete type.
	writeResponse(resp, contentType, http.StatusOK, &types.Empty{})
}

func readAndCloseBody(resp http.ResponseWriter, req *http.Request, contentType string) ([]byte, bool) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		writeError(resp, contentType, err, http.StatusBadRequest)
		return nil, false
	}
	if err = req.Body.Close(); err != nil {
		writeError(resp, contentType, err, http.StatusBadRequest)
		return nil, false
	}
	return body, true
}

// writeError encodes the HTTP error inside a rpc.Status message as required by the OTLP protocol.
func writeError(w http.ResponseWriter, contentType string, err error, statusCode int) {
	s, ok := status.FromError(err)
	if ok {
		writeResponse(w, contentType, statusCode, s.Proto())
	} else {
		writeErrorMsg(w, contentType, err.Error(), statusCode)
	}
}

// writeErrorMsg encodes the HTTP error message inside a rpc.Status message as required
// by the OTLP protocol.
func writeErrorMsg(w http.ResponseWriter, contentType string, errMsg string, statusCode int) {
	var s *status.Status
	if statusCode == http.StatusBadRequest {
		s = status.New(codes.InvalidArgument, errMsg)
	} else {
		s = status.New(codes.Unknown, errMsg)
	}

	writeResponse(w, contentType, statusCode, s.Proto())
}

// errorHandler encodes the HTTP error message inside a rpc.Status message as required
// by the OTLP protocol.
func errorHandler(w http.ResponseWriter, r *http.Request, errMsg string, statusCode int) {
	writeErrorMsg(w, r.Header.Get("Content-Type"), errMsg, statusCode)
}

// Pre-computed status with code=Internal to be used in case of a marshaling error.
var fallbackMsg = []byte(`{"code": 13, "message": "failed to marshal error message"}`)

const fallbackContentType = "application/json"

func writeResponse(w http.ResponseWriter, contentType string, statusCode int, rsp proto.Message) {
	var err error
	var msg []byte
	if contentType == "application/json" {
		buf := new(bytes.Buffer)
		err = jsonMarshaler.Marshal(buf, rsp)
		msg = buf.Bytes()
	} else {
		msg, err = proto.Marshal(rsp)
	}

	if err != nil {
		msg = fallbackMsg
		contentType = fallbackContentType
		statusCode = http.StatusInternalServerError
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	// Nothing we can do with the error if we cannot write to the response.
	_, _ = w.Write(msg)
}
