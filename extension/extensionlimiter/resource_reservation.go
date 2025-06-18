// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionlimiter // import "go.opentelemetry.io/collector/extension/extensionlimiter"

// ResourceReservation is modeled on pkg.go.dev/golang.org/x/time/rate#Reservation
// without the time dimension.
type ResourceReservation interface {
	// Delay returns a channel that will be closed when the
	// reservation request is granted. This resembles a context
	// Done channel.
	Delay() <-chan struct{}

	// Release is called after finishing with the reservation,
	// whether the delay has been reached or not.
	Release()
}

// ReleaseFunc is called when resources have been released after use.
//
// RelaseFunc values are never nil values, even in the error case for
// safety. Users typically will immediately defer a call to this.
type ReleaseFunc func()

// Release calls this function.
func (f ReleaseFunc) Release() {
	if f == nil {
		return
	}
	f()
}

// DelayFunc returns a channel that is closed when the request is
// permitted to go ahead.
type DelayFunc func() <-chan struct{}

// Delay calls this function.
func (f DelayFunc) Delay() <-chan struct{} {
	if f == nil {
		return immediateChan
	}
	return f()
}

// immediateChan is a singleton channel, already closed, used when a DelayFunc is nil.
var immediateChan = func() <-chan struct{} {
	ic := make(chan struct{})
	close(ic)
	return ic
}()

// resourceReservationImpl is a struct that implements ResourceReservation.
// The zero state is a no-op.
type resourceReservationImpl struct {
	DelayFunc
	ReleaseFunc
}

var _ ResourceReservation = resourceReservationImpl{}

// NewResourceReservationImpl returns a functional implementation of
// ResourceReservation.  Use a nil argument for the no-op implementation.
func NewResourceReservationImpl(df DelayFunc, rf ReleaseFunc) ResourceReservation {
	return resourceReservationImpl{
		DelayFunc: df,
		ReleaseFunc: rf,
	}
}
