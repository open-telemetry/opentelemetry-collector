// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// This semantic convention defines the attributes used to represent a feature flag evaluation as an event.
const (
	// The unique identifier of the feature flag.
	//
	// Type: string
	// Requirement Level: Required
	// Stability: stable
	// Examples: 'logo-color'
	AttributeFeatureFlagKey = "feature_flag.key"
	// The name of the service provider that performs the flag evaluation.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: stable
	// Examples: 'Flag Manager'
	AttributeFeatureFlagProviderName = "feature_flag.provider_name"
	// SHOULD be a semantic identifier for a value. If one is unavailable, a
	// stringified version of the value can be used.
	//
	// Type: string
	// Requirement Level: Recommended
	// Stability: stable
	// Examples: 'red', 'true', 'on'
	// Note: A semantic identifier, commonly referred to as a variant, provides a
	// means
	// for referring to a value without including the value itself. This can
	// provide additional context for understanding the meaning behind a value.
	// For example, the variant red maybe be used for the value #c05543.A stringified
	// version of the value can be used in situations where a
	// semantic identifier is unavailable. String representation of the value
	// should be determined by the implementer.
	AttributeFeatureFlagVariant = "feature_flag.variant"
)

// RPC received/sent message.
const (
	// Whether this is a received or sent message.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: stable
	AttributeMessageType = "message.type"
	// MUST be calculated as two different counters starting from 1 one for sent
	// messages and one for received message.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	// Note: This way we guarantee that the values will be consistent between
	// different implementations.
	AttributeMessageID = "message.id"
	// Compressed size of the message in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	AttributeMessageCompressedSize = "message.compressed_size"
	// Uncompressed size of the message in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: stable
	AttributeMessageUncompressedSize = "message.uncompressed_size"
)

const (
	// sent
	AttributeMessageTypeSent = "SENT"
	// received
	AttributeMessageTypeReceived = "RECEIVED"
)

// This document defines the attributes used to report a single exception associated with a span.
const (
	// SHOULD be set to true if the exception event is recorded at a point where it is
	// known that the exception is escaping the scope of the span.
	//
	// Type: boolean
	// Requirement Level: Optional
	// Stability: stable
	// Note: An exception is considered to have escaped (or left) the scope of a span,
	// if that span is ended while the exception is still logically &quot;in
	// flight&quot;.
	// This may be actually &quot;in flight&quot; in some languages (e.g. if the
	// exception
	// is passed to a Context manager's __exit__ method in Python) but will
	// usually be caught at the point of recording the exception in most languages.It
	// is usually not possible to determine at the point where an exception is thrown
	// whether it will escape the scope of a span.
	// However, it is trivial to know that an exception
	// will escape, if one checks for an active exception just before ending the span,
	// as done in the example above.It follows that an exception may still escape the
	// scope of the span
	// even if the exception.escaped attribute was not set or set to false,
	// since the event might have been recorded at a time where it was not
	// clear whether the exception will escape.
	AttributeExceptionEscaped = "exception.escaped"
)

func GetEventSemanticConventionAttributeNames() []string {
	return []string{
		AttributeFeatureFlagKey,
		AttributeFeatureFlagProviderName,
		AttributeFeatureFlagVariant,
		AttributeMessageType,
		AttributeMessageID,
		AttributeMessageCompressedSize,
		AttributeMessageUncompressedSize,
		AttributeExceptionEscaped,
	}
}
