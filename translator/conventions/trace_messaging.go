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

// OpenTelemetry Semantic Convention attribute names for messaging system related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/messaging.md
const (
	AttributeMessagingConversationID        = "messaging.conversation_id"
	AttributeMessagingDestination           = "messaging.destination"
	AttributeMessagingDestinationKind       = "messaging.destination_kind"
	AttributeMessagingMessageID             = "messaging.message_id"
	AttributeMessagingOperation             = "messaging.operation"
	AttributeMessagingPayloadCompressedSize = "messaging.message_payload_compressed_size_bytes"
	AttributeMessagingPayloadSize           = "messaging.message_payload_size_bytes"
	AttributeMessagingProtocol              = "messaging.protocol"
	AttributeMessagingProtocolVersion       = "messaging.protocol_version"
	AttributeMessagingSystem                = "messaging.system"
	AttributeMessagingTempDestination       = "messaging.temp_destination"
	AttributeMessagingURL                   = "messaging.url"
)
