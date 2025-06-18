package obsconsumer

type downstreamError struct {
	inner error
}

var _ error = downstreamError{}

func (de downstreamError) Error() string {
	return de.inner.Error()
}

func MarkAsDownstream(err error) error {
	return downstreamError{
		inner: err,
	}
}

func IsDownstream(err error) bool {
	if _, ok := err.(downstreamError); ok {
		return true
	}
	if wrapper, ok := err.(interface{ Unwrap() error }); ok {
		return IsDownstream(wrapper.Unwrap())
	}
	if wrapper, ok := err.(interface{ Unwrap() []error }); ok {
		for _, suberr := range wrapper.Unwrap() {
			if !IsDownstream(suberr) {
				return false
			}
		}
		return true
	}
	return false
}
