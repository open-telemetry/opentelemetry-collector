// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package test17

import "go.opentelemetry.io/collector/cmd/schemagen/internal/testdata/test17/internal"

// Handler is a named interface type.
type Handler interface {
	Handle() error
}

// Names is a named slice of strings.
type Names []string

// Groups is a named map of a named slice type.
type Groups map[string]Names

// Tree is a named slice type that refers to itself.
type Tree []Tree

type NamedTypesConfig struct {
	Handler Handler        `mapstructure:"handler"`
	Codec   internal.Codec `mapstructure:"codec"`
	Names   Names          `mapstructure:"names"`
	Groups  Groups         `mapstructure:"groups"`
	Tree    Tree           `mapstructure:"tree"`
	Extra   Groups         `mapstructure:",remain"`
}
