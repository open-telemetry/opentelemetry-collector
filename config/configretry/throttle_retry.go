package configretry

import (
	"time"
)

// TODO: Clean this by forcing all exporters to return an internal error type that always include the information about retries.
type throttleRetry struct {
	err   error
	delay time.Duration
}

func (t throttleRetry) Error() string {
	return "Throttle (" + t.delay.String() + "), error: " + t.err.Error()
}

func (t throttleRetry) Unwrap() error {
	return t.err
}

// NewThrottleRetryError creates a new throttle retry error.
func NewThrottleRetryError(err error, delay time.Duration) error {
	return throttleRetry{
		err:   err,
		delay: delay,
	}
}
