// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

// Config defines the configuration for a component.Component.
//
// Implementations and/or any sub-configs (other types embedded or included in the Config implementation)
// MUST implement confmap.Validator if any validation is required for that part of the configuration
// (e.g. check if a required field is present).
//
// A valid implementation MUST pass the check componenttest.CheckConfigStruct (return nil error).
type Config any

// ExtensionConfigResolver looks up a configured extension by its full ID.
// Callers must not modify the returned configuration.
type ExtensionConfigResolver func(ID) (Config, bool)

// ExtensionConfigValidator is an optional interface for receiver configurations
// that need to validate references to extension configurations before startup.
type ExtensionConfigValidator interface {
	ValidateWithExtensions(ExtensionConfigResolver) error
}
