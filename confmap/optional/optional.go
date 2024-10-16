package optional

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/confmap"
)

var _ confmap.Unmarshaler = (*Optional[any])(nil)

type Optional[T any] struct {
	value    T
	hasValue bool
}

func (o *Optional[T]) HasValue() bool {
	return o.hasValue
}

func (o *Optional[T]) Value() T {
	return o.value
}

func WithDefault[T any](val T) (o Optional[T]) {
	o.hasValue = true
	o.value = val
	return
}

func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	o.hasValue = true
	if err := conf.Unmarshal(&o.value); err != nil {
		return err
	}
	return nil
}

func (o *Optional[T]) UnmarshalPrimitive(val any) error {
	o.hasValue = true
	valT, ok := val.(T)
	if !ok {
		return fmt.Errorf("cannot cast val to %v", reflect.TypeOf(o.value))
	}
	o.value = valT
	return nil
}
