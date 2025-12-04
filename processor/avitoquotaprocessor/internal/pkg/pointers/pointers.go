package pointers

import "time"

func ToString(s *string, def string) string {
	if s != nil {
		return *s
	}
	return def
}

func ToInt(i *int, def int) int {
	if i != nil {
		return *i
	}
	return def
}

func ToInt32(i *int32, def int32) int32 {
	if i != nil {
		return *i
	}
	return def
}

func ToInt64(i *int64, def int64) int64 {
	if i != nil {
		return *i
	}
	return def
}

func ToBool(b *bool, def bool) bool {
	if b != nil {
		return *b
	}
	return def
}

func ToTime(v *time.Time, def time.Time) time.Time {
	if v != nil {
		return *v
	}
	return def
}

func ToFloat32(f *float32, def float32) float32 {
	if f != nil {
		return *f
	}
	return def
}

func ToFloat64(f *float64, def float64) float64 {
	if f != nil {
		return *f
	}
	return def
}

func FromString(str string) *string {
	return &str
}

func FromInt(number int) *int {
	return &number
}

func FromInt32(number int32) *int32 {
	return &number
}

func FromInt64(number int64) *int64 {
	return &number
}

func FromFloat32(number float32) *float32 {
	return &number
}

func FromFloat64(number float64) *float64 {
	return &number
}

func FromTime(time time.Time) *time.Time {
	return &time
}

func FromBool(bool bool) *bool {
	return &bool
}

func From[T any](t T) *T {
	return &t
}

func OmitEmptyString(ptr string) *string {
	if ptr == "" {
		return nil
	}
	return &ptr
}
