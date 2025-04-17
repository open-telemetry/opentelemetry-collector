// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"

	"go.opentelemetry.io/collector/confmap"
)

type (
	LogName string
)

func (ln LogName) Render() (string, error) {
	return FormatIdentifier(string(ln), true)
}

func (ln LogName) RenderUnexported() (string, error) {
	return FormatIdentifier(string(ln), false)
}

type Log struct {
	// Enabled defines whether the log is enabled by default.
	Enabled bool `mapstructure:"enabled"`

	// Warnings that will be shown to user under specified conditions.
	Warnings Warnings `mapstructure:"warnings"`

	// Body stores metadata for log body
	Body *Body `mapstructure:"body"`

	// Description of the log.
	Description string `mapstructure:"description"`

	// The stability level of the log.
	Stability Stability `mapstructure:"stability"`

	// ExtendedDocumentation of the log. If specified, this will
	// be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Attributes is the list of attributes that the log emits.
	Attributes []AttributeName `mapstructure:"attributes"`
}

func (l *Log) validate() error {
	var errs error
	if l.Description == "" {
		errs = errors.Join(errs, errors.New(`missing log description`))
	}
	return errs
}

func (l *Log) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("enabled") {
		return errors.New("missing required field: `enabled`")
	}
	return parser.Unmarshal(l)
}

type Body struct {
	Type ValueType `mapstructure:"type"`
}
