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
	"net/http"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// xProtobufMarshaler is a Marshaler which wraps runtime.ProtoMarshaller
// and sets ContentType to application/x-protobuf
type xProtobufMarshaler struct {
	*runtime.ProtoMarshaller
}

// ContentType always returns "application/x-protobuf".
func (*xProtobufMarshaler) ContentType() string {
	return "application/x-protobuf"
}

var jsonMarshaller = &jsonpb.Marshaler{}

// errorHandler encodes the HTTP error message inside a rpc.Status message as required
// by the OTLP protocol.
func errorHandler(w http.ResponseWriter, r *http.Request, errMsg string, statusCode int) {
	var (
		msg []byte
		s   *status.Status
		err error
	)
	// Pre-computed status with code=Internal to be used in case of a marshaling error.
	fallbackMsg := []byte(`{"code": 13, "message": "failed to marshal error message"}`)
	fallbackContentType := "application/json"

	if statusCode == http.StatusBadRequest {
		s = status.New(codes.InvalidArgument, errMsg)
	} else {
		s = status.New(codes.Internal, errMsg)
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		buf := new(bytes.Buffer)
		err = jsonMarshaller.Marshal(buf, s.Proto())
		msg = buf.Bytes()
	} else {
		msg, err = proto.Marshal(s.Proto())
	}
	if err != nil {
		msg = fallbackMsg
		contentType = fallbackContentType
		statusCode = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	w.Write(msg)
}
