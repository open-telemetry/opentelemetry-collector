// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/mitchellh/mapstructure"

	encoder "go.opentelemetry.io/collector/confmap/internal/mapstructure"
)

const (
	// KeyDelimiter is used as the default key delimiter in the default koanf instance.
	KeyDelimiter = "::"
)

// New creates a new empty confmap.Conf instance.
func New() *Conf {
	return &Conf{k: koanf.New(KeyDelimiter)}
}

// NewFromStringMap creates a confmap.Conf from a map[string]any.
func NewFromStringMap(data map[string]any) *Conf {
	p := New()
	// Cannot return error because the koanf instance is empty.
	_ = p.k.Load(confmap.Provider(data, KeyDelimiter), nil)
	return p
}

// Conf represents the raw configuration map for the OpenTelemetry Collector.
// The confmap.Conf can be unmarshalled into the Collector's config using the "service" package.
type Conf struct {
	k *koanf.Koanf
}

// AllKeys returns all keys holding a value, regardless of where they are set.
// Nested keys are returned with a KeyDelimiter separator.
func (l *Conf) AllKeys() []string {
	return l.k.Keys()
}

type UnmarshalOption interface {
	apply(*unmarshalOption)
}

type unmarshalOption struct {
	errorUnused bool
}

// WithErrorUnused sets an option to error when there are existing
// keys in the original Conf that were unused in the decoding process
// (extra keys).
func WithErrorUnused() UnmarshalOption {
	return unmarshalOptionFunc(func(uo *unmarshalOption) {
		uo.errorUnused = true
	})
}

type unmarshalOptionFunc func(*unmarshalOption)

func (fn unmarshalOptionFunc) apply(set *unmarshalOption) {
	fn(set)
}

// Unmarshal unmarshalls the config into a struct using the given options.
// Tags on the fields of the structure must be properly set.
func (l *Conf) Unmarshal(result any, opts ...UnmarshalOption) error {
	set := unmarshalOption{}
	for _, opt := range opts {
		opt.apply(&set)
	}
	return decodeConfig(l, result, set.errorUnused)
}

type marshalOption struct{}

type MarshalOption interface {
	apply(*marshalOption)
}

// Marshal encodes the config and merges it into the Conf.
func (l *Conf) Marshal(rawVal any, _ ...MarshalOption) error {
	enc := encoder.New(encoderConfig(rawVal))
	data, err := enc.Encode(rawVal)
	if err != nil {
		return err
	}
	out, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config encoding")
	}
	return l.Merge(NewFromStringMap(out))
}

// Get can retrieve any value given the key to use.
func (l *Conf) Get(key string) any {
	return l.k.Get(key)
}

// IsSet checks to see if the key has been set in any of the data locations.
func (l *Conf) IsSet(key string) bool {
	return l.k.Exists(key)
}

// Merge merges the input given configuration into the existing config.
// Note that the given map may be modified.
func (l *Conf) Merge(in *Conf) error {
	return l.k.Merge(in.k)
}

// Sub returns new Conf instance representing a sub-config of this instance.
// It returns an error is the sub-config is not a map[string]any (use Get()), and an empty Map if none exists.
func (l *Conf) Sub(key string) (*Conf, error) {
	// Code inspired by the koanf "Cut" func, but returns an error instead of empty map for unsupported sub-config type.
	data := l.Get(key)
	if data == nil {
		return New(), nil
	}

	if v, ok := data.(map[string]any); ok {
		return NewFromStringMap(v), nil
	}

	return nil, fmt.Errorf("unexpected sub-config value kind for key:%s value:%v kind:%v)", key, data, reflect.TypeOf(data).Kind())
}

// ToStringMap creates a map[string]any from a Parser.
func (l *Conf) ToStringMap() map[string]any {
	return maps.Unflatten(l.k.All(), KeyDelimiter)
}

// decodeConfig decodes the contents of the Conf into the result argument, using a
// mapstructure decoder with the following notable behaviors. Ensures that maps whose
// values are nil pointer structs resolved to the zero value of the target struct (see
// expandNilStructPointers). Converts string to []string by splitting on ','. Ensures
// uniqueness of component IDs (see mapKeyStringToMapKeyTextUnmarshalerHookFunc).
// Decodes time.Duration from strings. Allows custom unmarshaling for structs implementing
// encoding.TextUnmarshaler. Allows custom unmarshaling for structs implementing confmap.Unmarshaler.
func decodeConfig(m *Conf, result any, errorUnused bool) error {
	dc := &mapstructure.DecoderConfig{
		ErrorUnused:      errorUnused,
		Result:           result,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
		MatchName:        caseSensitiveMatchName,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			expandNilStructPointersHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapKeyStringToMapKeyTextUnmarshalerHookFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
			unmarshalerHookFunc(result),
		),
	}
	decoder, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	return decoder.Decode(m.ToStringMap())
}

// encoderConfig returns a default encoder.EncoderConfig that includes
// an EncodeHook that handles both TextMarshaller and Marshaler
// interfaces.
func encoderConfig(rawVal any) *encoder.EncoderConfig {
	return &encoder.EncoderConfig{
		EncodeHook: mapstructure.ComposeDecodeHookFunc(
			encoder.TextMarshalerHookFunc(),
			marshalerHookFunc(rawVal),
		),
	}
}

// case-sensitive version of the callback to be used in the MatchName property
// of the DecoderConfig. The default for MatchEqual is to use strings.EqualFold,
// which is case-insensitive.
func caseSensitiveMatchName(a, b string) bool {
	return a == b
}

// In cases where a config has a mapping of something to a struct pointers
// we want nil values to resolve to a pointer to the zero value of the
// underlying struct just as we want nil values of a mapping of something
// to a struct to resolve to the zero value of that struct.
//
// e.g. given a config type:
// type Config struct { Thing *SomeStruct `mapstructure:"thing"` }
//
// and yaml of:
// config:
//
//	thing:
//
// we want an unmarshaled Config to be equivalent to
// Config{Thing: &SomeStruct{}} instead of Config{Thing: nil}
func expandNilStructPointersHookFunc() mapstructure.DecodeHookFuncValue {
	return func(from reflect.Value, to reflect.Value) (any, error) {
		// ensure we are dealing with map to map comparison
		if from.Kind() == reflect.Map && to.Kind() == reflect.Map {
			toElem := to.Type().Elem()
			// ensure that map values are pointers to a struct
			// (that may be nil and require manual setting w/ zero value)
			if toElem.Kind() == reflect.Ptr && toElem.Elem().Kind() == reflect.Struct {
				fromRange := from.MapRange()
				for fromRange.Next() {
					fromKey := fromRange.Key()
					fromValue := fromRange.Value()
					// ensure that we've run into a nil pointer instance
					if fromValue.IsNil() {
						newFromValue := reflect.New(toElem.Elem())
						from.SetMapIndex(fromKey, newFromValue)
					}
				}
			}
		}
		return from.Interface(), nil
	}
}

// mapKeyStringToMapKeyTextUnmarshalerHookFunc returns a DecodeHookFuncType that checks that a conversion from
// map[string]any to map[encoding.TextUnmarshaler]any does not overwrite keys,
// when UnmarshalText produces equal elements from different strings (e.g. trims whitespaces).
//
// This is needed in combination with ComponentID, which may produce equal IDs for different strings,
// and an error needs to be returned in that case, otherwise the last equivalent ID overwrites the previous one.
func mapKeyStringToMapKeyTextUnmarshalerHookFunc() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.Map || f.Key().Kind() != reflect.String {
			return data, nil
		}

		if t.Kind() != reflect.Map {
			return data, nil
		}

		if _, ok := reflect.New(t.Key()).Interface().(encoding.TextUnmarshaler); !ok {
			return data, nil
		}

		m := reflect.MakeMap(reflect.MapOf(t.Key(), reflect.TypeOf(true)))
		for k := range data.(map[string]any) {
			tKey := reflect.New(t.Key())
			if err := tKey.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(k)); err != nil {
				return nil, err
			}

			if m.MapIndex(reflect.Indirect(tKey)).IsValid() {
				return nil, fmt.Errorf("duplicate name %q after unmarshaling %v", k, tKey)
			}
			m.SetMapIndex(reflect.Indirect(tKey), reflect.ValueOf(true))
		}
		return data, nil
	}
}

// Provides a mechanism for individual structs to define their own unmarshal logic,
// by implementing the Unmarshaler interface.
func unmarshalerHookFunc(result any) mapstructure.DecodeHookFuncValue {
	return func(from reflect.Value, to reflect.Value) (any, error) {
		if !to.CanAddr() {
			return from.Interface(), nil
		}

		toPtr := to.Addr().Interface()
		// Need to ignore the top structure to avoid circular dependency.
		if toPtr == result {
			return from.Interface(), nil
		}

		unmarshaler, ok := toPtr.(Unmarshaler)
		if !ok {
			return from.Interface(), nil
		}

		if _, ok = from.Interface().(map[string]any); !ok {
			return from.Interface(), nil
		}

		// Use the current object if not nil (to preserve other configs in the object), otherwise zero initialize.
		if to.Addr().IsNil() {
			unmarshaler = reflect.New(to.Type()).Interface().(Unmarshaler)
		}

		if err := unmarshaler.Unmarshal(NewFromStringMap(from.Interface().(map[string]any))); err != nil {
			return nil, err
		}

		return unmarshaler, nil
	}
}

// marshalerHookFunc returns a DecodeHookFuncValue that checks structs that aren't
// the original to see if they implement the Marshaler interface.
func marshalerHookFunc(orig any) mapstructure.DecodeHookFuncValue {
	origType := reflect.TypeOf(orig)
	return func(from reflect.Value, _ reflect.Value) (any, error) {
		if from.Kind() != reflect.Struct {
			return from.Interface(), nil
		}

		// ignore original to avoid infinite loop.
		if from.Type() == origType && reflect.DeepEqual(from.Interface(), orig) {
			return from.Interface(), nil
		}
		marshaler, ok := from.Interface().(Marshaler)
		if !ok {
			return from.Interface(), nil
		}
		conf := New()
		if err := marshaler.Marshal(conf); err != nil {
			return nil, err
		}
		return conf.ToStringMap(), nil
	}
}

// Unmarshaler interface may be implemented by types to customize their behavior when being unmarshaled from a Conf.
type Unmarshaler interface {
	// Unmarshal a Conf into the struct in a custom way.
	// The Conf for this specific component may be nil or empty if no config available.
	Unmarshal(component *Conf) error
}

// Marshaler defines an optional interface for custom configuration marshaling.
// A configuration struct can implement this interface to override the default
// marshaling.
type Marshaler interface {
	// Marshal the config into a Conf in a custom way.
	// The Conf will be empty and can be merged into.
	Marshal(component *Conf) error
}
