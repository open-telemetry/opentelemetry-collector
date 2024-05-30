package confmap

type Optional[T any] struct {
	hasValue bool
	value    T
}

var _ Unmarshaler = (*Optional[any])(nil)

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func (o Optional[T]) HasValue() bool {
	return o.hasValue
}

func (o Optional[T]) Value() T {
	return o.value
}

func (o *Optional[T]) Unmarshal(conf *Conf) error {
	if err := conf.Unmarshal(&o.value); err != nil {
		return err
	}
	o.hasValue = true
	return nil
}
