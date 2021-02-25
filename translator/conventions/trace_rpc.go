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

package conventions

// OpenTelemetry Semantic Convention attribute names for gRPC related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/rpc.md
const (
	AttributeMessageCompressedSize   = "message.compressed_size"
	AttributeMessageID               = "message.id"
	AttributeMessageType             = "message.type"
	AttributeMessageUncompressedSize = "message.uncompressed_size"
	AttributeRPCMethod               = "rpc.method"
	AttributeRPCService              = "rpc.service"
	AttributeRPCSystem               = "rpc.system"
	EventTypeMessage                 = "message"
	MessageTypeReceived              = "RECEIVED"
	MessageTypeSent                  = "SENT"
)
