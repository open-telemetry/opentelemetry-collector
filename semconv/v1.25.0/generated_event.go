// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Code generated from semantic convention specification. DO NOT EDIT.

package semconv

// This event represents an occurrence of a lifecycle transition on the iOS
// platform.
const (
	// This attribute represents the state the application has transitioned into at
	// the occurrence of the event.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Note: The iOS lifecycle states are defined in the UIApplicationDelegate
	// documentation, and from which the OS terminology column values are derived.
	AttributeIosState = "ios.state"
)

const (
	// The app has become `active`. Associated with UIKit notification `applicationDidBecomeActive`
	AttributeIosStateActive = "active"
	// The app is now `inactive`. Associated with UIKit notification `applicationWillResignActive`
	AttributeIosStateInactive = "inactive"
	// The app is now in the background. This value is associated with UIKit notification `applicationDidEnterBackground`
	AttributeIosStateBackground = "background"
	// The app is now in the foreground. This value is associated with UIKit notification `applicationWillEnterForeground`
	AttributeIosStateForeground = "foreground"
	// The app is about to terminate. Associated with UIKit notification `applicationWillTerminate`
	AttributeIosStateTerminate = "terminate"
)

// This event represents an occurrence of a lifecycle transition on the Android
// platform.
const (
	// This attribute represents the state the application has transitioned into at
	// the occurrence of the event.
	//
	// Type: Enum
	// Requirement Level: Required
	// Stability: experimental
	// Note: The Android lifecycle states are defined in Activity lifecycle callbacks,
	// and from which the OS identifiers are derived.
	AttributeAndroidState = "android.state"
)

const (
	// Any time before Activity.onResume() or, if the app has no Activity, Context.startService() has been called in the app for the first time
	AttributeAndroidStateCreated = "created"
	// Any time after Activity.onPause() or, if the app has no Activity, Context.stopService() has been called when the app was in the foreground state
	AttributeAndroidStateBackground = "background"
	// Any time after Activity.onResume() or, if the app has no Activity, Context.startService() has been called when the app was in either the created or background states
	AttributeAndroidStateForeground = "foreground"
)

// RPC received/sent message.
const (
	// Compressed size of the message in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessageCompressedSize = "message.compressed_size"
	// MUST be calculated as two different counters starting from 1 one for sent
	// messages and one for received message.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	// Note: This way we guarantee that the values will be consistent between
	// different implementations.
	AttributeMessageID = "message.id"
	// Whether this is a received or sent message.
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessageType = "message.type"
	// Uncompressed size of the message in bytes.
	//
	// Type: int
	// Requirement Level: Optional
	// Stability: experimental
	AttributeMessageUncompressedSize = "message.uncompressed_size"
)

const (
	// sent
	AttributeMessageTypeSent = "SENT"
	// received
	AttributeMessageTypeReceived = "RECEIVED"
)

func GetEventSemanticConventionAttributeNames() []string {
	return []string{
		AttributeIosState,
		AttributeAndroidState,
		AttributeMessageCompressedSize,
		AttributeMessageID,
		AttributeMessageType,
		AttributeMessageUncompressedSize,
	}
}
