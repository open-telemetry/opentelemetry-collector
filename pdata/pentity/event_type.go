// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pentity // import "go.opentelemetry.io/collector/pdata/pentity"

// EventType specifies the type of entity event.
type EventType int32

const (
	// EventTypeEmpty means that metric type is unset.
	EventTypeEmpty EventType = iota
	EventTypeEntityState
	EventTypeEntityDelete
)

// String returns the string representation of the EventType.
func (mdt EventType) String() string {
	switch mdt {
	case EventTypeEmpty:
		return "Empty"
	case EventTypeEntityState:
		return "State"
	case EventTypeEntityDelete:
		return "Delete"
	}
	return ""
}
