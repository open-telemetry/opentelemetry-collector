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

package kafkareceiver

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Unmarshaller deserializes the message body.
type Unmarshaller interface {
	// Unmarshal deserializes the message body into traces.
	Unmarshal([]byte) (pdata.Traces, error)

	// Encoding of the serialized messages.
	Encoding() string
}

var unmarshallers = map[string]Unmarshaller{}

// GetUnmarshaller returns unmarshaller for given encoding or nil if not found.
func GetUnmarshaller(encoding string) Unmarshaller {
	return unmarshallers[encoding]
}

// RegisterUnmarshaller registers unmarshaller.
func RegisterUnmarshaller(unmarshaller Unmarshaller) {
	unmarshallers[unmarshaller.Encoding()] = unmarshaller
}
