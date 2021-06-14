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

package configcheck

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// The regular expression for valid config field tag.
var configFieldTagRegExp = regexp.MustCompile("^[a-z0-9][a-z0-9_]*$")

// ValidateConfigFromFactories checks if all configurations for the given factories
// are satisfying the patterns used by the collector.
func ValidateConfigFromFactories(factories component.Factories) error {
	var errs []error

	for _, factory := range factories.Receivers {
		if err := ValidateConfig(factory.CreateDefaultConfig()); err != nil {
			errs = append(errs, err)
		}
	}
	for _, factory := range factories.Processors {
		if err := ValidateConfig(factory.CreateDefaultConfig()); err != nil {
			errs = append(errs, err)
		}
	}
	for _, factory := range factories.Exporters {
		if err := ValidateConfig(factory.CreateDefaultConfig()); err != nil {
			errs = append(errs, err)
		}
	}
	for _, factory := range factories.Extensions {
		if err := ValidateConfig(factory.CreateDefaultConfig()); err != nil {
			errs = append(errs, err)
		}
	}

	return consumererror.Combine(errs)
}

// ValidateConfig enforces that given configuration object is following the patterns
// used by the collector. This ensures consistency between different implementations
// of components and extensions. It is recommended for implementers of components
// to call this function on their tests passing the default configuration of the
// component factory.
func ValidateConfig(config interface{}) error {
	t := reflect.TypeOf(config)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct or a pointer to one, the passed object is a %s", t.Kind())
	}

	return validateConfigDataType(t)
}

// validateConfigDataType performs a descending validation of the given type.
// If the type is a struct it goes to each of its fields to check for the proper
// tags.
func validateConfigDataType(t reflect.Type) error {
	var errs []error

	switch t.Kind() {
	case reflect.Ptr:
		if err := validateConfigDataType(t.Elem()); err != nil {
			errs = append(errs, err)
		}
	case reflect.Struct:
		// Reflect on the pointed data and check each of its fields.
		nf := t.NumField()
		for i := 0; i < nf; i++ {
			f := t.Field(i)
			if err := checkStructFieldTags(f); err != nil {
				errs = append(errs, err)
			}
		}
	default:
		// The config object can carry other types but they are not used when
		// reading the configuration via koanf so ignore them. Basically ignore:
		// reflect.Uintptr, reflect.Chan, reflect.Func, reflect.Interface, and
		// reflect.UnsafePointer.
	}

	if err := consumererror.Combine(errs); err != nil {
		return fmt.Errorf(
			"type %q from package %q has invalid config settings: %v",
			t.Name(),
			t.PkgPath(),
			err)
	}

	return nil
}

// checkStructFieldTags inspects the tags of a struct field.
func checkStructFieldTags(f reflect.StructField) error {

	tagValue := f.Tag.Get("mapstructure")
	if tagValue == "" {

		// Ignore special types.
		switch f.Type.Kind() {
		case reflect.Interface, reflect.Chan, reflect.Func, reflect.Uintptr, reflect.UnsafePointer:
			// Allow the config to carry the types above, but since they are not read
			// when loading configuration, just ignore them.
			return nil
		}

		// Public fields of other types should be tagged.
		chars := []byte(f.Name)
		if len(chars) > 0 && chars[0] >= 'A' && chars[0] <= 'Z' {
			return fmt.Errorf("mapstructure tag not present on field %q", f.Name)
		}

		// Not public field, no need to have a tag.
		return nil
	}

	tagParts := strings.Split(tagValue, ",")
	if tagParts[0] != "" {
		if tagParts[0] == "-" {
			// Nothing to do, as mapstructure decode skips this field.
			return nil
		}
	}

	// Check if squash is specified.
	squash := false
	for _, tag := range tagParts[1:] {
		if tag == "squash" {
			squash = true
			break
		}
	}

	if squash {
		// Field was squashed.
		if (f.Type.Kind() != reflect.Struct) && (f.Type.Kind() != reflect.Ptr || f.Type.Elem().Kind() != reflect.Struct) {
			return fmt.Errorf(
				"attempt to squash non-struct type on field %q", f.Name)
		}
	}

	switch f.Type.Kind() {
	case reflect.Struct:
		// It is another struct, continue down-level.
		return validateConfigDataType(f.Type)

	case reflect.Map, reflect.Slice, reflect.Array:
		// The element of map, array, or slice can be itself a configuration object.
		return validateConfigDataType(f.Type.Elem())

	default:
		fieldTag := tagParts[0]
		if !configFieldTagRegExp.MatchString(fieldTag) {
			return fmt.Errorf(
				"field %q has config tag %q which doesn't satisfy %q",
				f.Name,
				fieldTag,
				configFieldTagRegExp.String())
		}
	}

	return nil
}
