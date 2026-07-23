// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package experr // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"

import (
	"errors"
	"fmt"
)

type shutdownErr struct {
	err error
}

func NewShutdownErr(err error) error {
	return shutdownErr{err: err}
}

func (s shutdownErr) Error() string {
	return "interrupted due to shutdown: " + s.err.Error()
}

func (s shutdownErr) Unwrap() error {
	return s.err
}

func IsShutdownErr(err error) bool {
	var sdErr shutdownErr
	return errors.As(err, &sdErr)
}

// DroppedItemsErr is a non-failure sentinel that an exporter can return to
// signal that it intentionally dropped a number of items rather than failing
// to send them.  The error is not propagated to the pipeline as a failure —
// it is used only to update the dropped-items telemetry counters.
type DroppedItemsErr struct {
	// Dropped is the number of items that were intentionally dropped.
	Dropped int
	// Reason is an optional human-readable description of why the items were
	// dropped (e.g. "incompatible temporality").
	Reason string
}

// NewDroppedItemsErr creates a DroppedItemsErr for the given count and reason.
func NewDroppedItemsErr(dropped int, reason string) error {
	return &DroppedItemsErr{Dropped: dropped, Reason: reason}
}

func (d *DroppedItemsErr) Error() string {
	if d.Reason != "" {
		return fmt.Sprintf("dropped %d items: %s", d.Dropped, d.Reason)
	}
	return fmt.Sprintf("dropped %d items", d.Dropped)
}

// DroppedItemsFromErr extracts the DroppedItemsErr from err, if present, and
// returns it together with a boolean indicating success.
func DroppedItemsFromErr(err error) (*DroppedItemsErr, bool) {
	var d *DroppedItemsErr
	if errors.As(err, &d) {
		return d, true
	}
	return nil, false
}
