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

package plogotlp // import "go.opentelemetry.io/collector/pdata/plog/plogotlp"

import (
	"bytes"

	"go.opentelemetry.io/collector/pdata/internal"
	otlpcollectorlog "go.opentelemetry.io/collector/pdata/internal/data/protogen/collector/logs/v1"
	"go.opentelemetry.io/collector/pdata/internal/otlp"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/internal/plogjson"
)

// Request represents the request for gRPC/HTTP client/server.
// It's a wrapper for plog.Logs data.
type Request struct {
	orig *otlpcollectorlog.ExportLogsServiceRequest
}

// NewRequest returns an empty Request.
func NewRequest() Request {
	return Request{orig: &otlpcollectorlog.ExportLogsServiceRequest{}}
}

// NewRequestFromLogs returns a Request from plog.Logs.
// Because Request is a wrapper for plog.Logs,
// any changes to the provided Logs struct will be reflected in the Request and vice versa.
func NewRequestFromLogs(ld plog.Logs) Request {
	return Request{orig: internal.GetOrigLogs(internal.Logs(ld))}
}

// MarshalProto marshals Request into proto bytes.
func (lr Request) MarshalProto() ([]byte, error) {
	return lr.orig.Marshal()
}

// UnmarshalProto unmarshalls Request from proto bytes.
func (lr Request) UnmarshalProto(data []byte) error {
	if err := lr.orig.Unmarshal(data); err != nil {
		return err
	}
	otlp.MigrateLogs(lr.orig.ResourceLogs)
	return nil
}

// MarshalJSON marshals Request into JSON bytes.
func (lr Request) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := plogjson.JSONMarshaler.Marshal(&buf, lr.orig); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshalls Request from JSON bytes.
func (lr Request) UnmarshalJSON(data []byte) error {
	return plogjson.UnmarshalExportLogsServiceRequest(data, lr.orig)
}

func (lr Request) Logs() plog.Logs {
	return plog.Logs(internal.NewLogs(lr.orig))
}
