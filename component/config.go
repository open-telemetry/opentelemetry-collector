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

package component // import "go.opentelemetry.io/collector/component"

import (
	"reflect"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/confmap"
)

// Config defines the configuration for a component.Component.
//
// Implementations and/or any sub-configs (other types embedded or included in the Config implementation)
// MUST implement the ConfigValidator if any validation is required for that part of the configuration
// (e.g. check if a required field is present).
//
// A valid implementation MUST pass the check componenttest.CheckConfigStruct (return nil error).
type Config any

// As interface types are only used for static typing, a common idiom to find the reflection Type
// for an interface type Foo is to use a *Foo value.
var configValidatorType = reflect.TypeOf((*ConfigValidator)(nil)).Elem()

// UnmarshalConfig helper function to UnmarshalConfig a Config.
// It checks if the config implements confmap.Unmarshaler and uses that if available,
// otherwise uses Map.UnmarshalExact, erroring if a field is nonexistent.
func UnmarshalConfig(conf *confmap.Conf, intoCfg Config) error {
	if cu, ok := intoCfg.(confmap.Unmarshaler); ok {
		return cu.Unmarshal(conf)
	}

	return conf.Unmarshal(intoCfg, confmap.WithErrorUnused())
}

// ConfigValidator defines an optional interface for configurations to implement to do validation.
type ConfigValidator interface {
	// Validate the configuration and returns an error if invalid.
	Validate() error
}

// ValidateConfig validates a config, by doing this:
//   - Call Validate on the config itself if the config implements ConfigValidator.
func ValidateConfig(cfg Config) error {
	return validate(reflect.ValueOf(cfg))
}

func validate(v reflect.Value) error {
	// Validate the value itself.
	switch v.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Ptr:
		return validate(v.Elem())
	case reflect.Struct:
		var errs error
		errs = multierr.Append(errs, callValidateIfPossible(v))
		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			errs = multierr.Append(errs, validate(v.Field(i)))
		}
		return errs
	case reflect.Slice, reflect.Array:
		var errs error
		errs = multierr.Append(errs, callValidateIfPossible(v))
		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.Len(); i++ {
			errs = multierr.Append(errs, validate(v.Index(i)))
		}
		return errs
	case reflect.Map:
		var errs error
		errs = multierr.Append(errs, callValidateIfPossible(v))
		iter := v.MapRange()
		for iter.Next() {
			errs = multierr.Append(errs, validate(iter.Key()))
			errs = multierr.Append(errs, validate(iter.Value()))
		}
		return errs
	default:
		return callValidateIfPossible(v)
	}
}

func callValidateIfPossible(v reflect.Value) error {
	// If the value type implements ConfigValidator just call Validate
	if v.Type().Implements(configValidatorType) {
		return v.Interface().(ConfigValidator).Validate()
	}

	// If the pointer type implements ConfigValidator call Validate on the pointer to the current value.
	if reflect.PtrTo(v.Type()).Implements(configValidatorType) {
		// If not addressable, then create a new *V pointer and set the value to current v.
		if !v.CanAddr() {
			pv := reflect.New(reflect.PtrTo(v.Type()).Elem())
			pv.Elem().Set(v)
			v = pv.Elem()
		}
		return v.Addr().Interface().(ConfigValidator).Validate()
	}

	return nil
}

// Type is the component type as it is used in the config.
type Type string

// DataType is a special Type that represents the data types supported by the collector. We currently support
// collecting metrics, traces and logs, this can expand in the future.
type DataType = Type

// Currently supported data types. Add new data types here when new types are supported in the future.
const (
	// DataTypeTraces is the data type tag for traces.
	DataTypeTraces DataType = "traces"

	// DataTypeMetrics is the data type tag for metrics.
	DataTypeMetrics DataType = "metrics"

	// DataTypeLogs is the data type tag for logs.
	DataTypeLogs DataType = "logs"
)
