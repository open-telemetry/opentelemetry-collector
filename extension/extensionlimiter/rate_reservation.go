// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

import (
	"time"
)

// RateReservation is modeled on pkg.go.dev/golang.org/x/time/rate#Reservation
type RateReservation interface {
	// WaitTime returns the duration until this reservation may
	// proceed. A typical implementation uses Reservation.DelayFrom(time.Now()),
	// and callers typically/ use time.After or time.Timer to implement a delay.
	WaitTime() time.Duration

	// Cancel cancels the reservation before it is used. A typical
	// implementation uses Reservation.CancelAt(time.Now()).
	Cancel()
}

// WaitTimeFunc is a functional way to construct WaitTime functions.
type WaitTimeFunc func() time.Duration

// WaitTime implements part of Reservation.
func (f WaitTimeFunc) WaitTime() time.Duration {
	if f == nil {
		return 0
	}
	return f.WaitTime()
}

// CancelFunc is a functional way to construct Cancel functions.
type CancelFunc func()

// Reserve implements part of the RateReserveer interface.
func (f CancelFunc) Cancel() {
	if f == nil {
		return
	}
	f.Cancel()
}

// rateReservationImpl is a struct that implements RateReservation.
// The zero state is a no-op.
type rateReservationImpl struct {
	WaitTimeFunc
	CancelFunc
}

var _ RateReservation = rateReservationImpl{}

// NewRateReservationImpl returns a functional implementation of
// RateReservation.  Use a nil argument for the no-op implementation.
func NewRateReservationImpl(wf WaitTimeFunc, cf CancelFunc) RateReservation {
	return rateReservationImpl{
		WaitTimeFunc: wf,
		CancelFunc: cf,
	}
}

