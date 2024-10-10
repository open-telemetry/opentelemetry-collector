package optional

import "go.opentelemetry.io/collector/confmap"

type Optional[T any] struct {
	Value    T
	HasValue bool
}

func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	o.HasValue = true
	if err := conf.Unmarshal(&o.Value); err != nil {
		return err
	}
	return nil
}
