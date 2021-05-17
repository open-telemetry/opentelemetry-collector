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

package encoding

import "fmt"

// Type is the encoding format that a model is serialized to.
type Type string

const (
	Protobuf Type = "protobuf"
	JSON     Type = "json"
	Thrift   Type = "thrift"
)

func (e Type) String() string {
	return string(e)
}

// ErrUnavailableEncoding is returned when the requested encoding is not present.
type ErrUnavailableEncoding struct {
	Encoding Type
}

func (e *ErrUnavailableEncoding) Error() string {
	return fmt.Sprintf("unsupported encoding %q", e.Encoding)
}
