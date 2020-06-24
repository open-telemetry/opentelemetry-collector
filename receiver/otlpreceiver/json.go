// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlpreceiver

import (
	"errors"
	"io"

	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

const jsonUnsupportedErrorMessage = "'application/json' encoding is not supported"

type jsonUnsupportedMarshaler struct {
	m *gatewayruntime.JSONPb
}

// returns a custom JSON handler which always returns error
func newJSONUnsupportedMarshaler() *jsonUnsupportedMarshaler {
	return &jsonUnsupportedMarshaler{
		m: &gatewayruntime.JSONPb{},
	}
}

func (j *jsonUnsupportedMarshaler) Marshal(v interface{}) ([]byte, error) { return j.m.Marshal(v) }
func (j *jsonUnsupportedMarshaler) Unmarshal(data []byte, v interface{}) error {
	return j.m.Unmarshal(data, v)
}
func (j *jsonUnsupportedMarshaler) NewDecoder(r io.Reader) gatewayruntime.Decoder {
	return &jsonUnsupportedDecoder{}
}
func (j *jsonUnsupportedMarshaler) NewEncoder(w io.Writer) gatewayruntime.Encoder {
	return j.m.NewEncoder(w)
}
func (j *jsonUnsupportedMarshaler) ContentType() string { return j.m.ContentType() }

type jsonUnsupportedDecoder struct {
}

func (d *jsonUnsupportedDecoder) Decode(v interface{}) error {
	return errors.New(jsonUnsupportedErrorMessage)
}
