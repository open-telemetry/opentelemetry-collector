// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package modfile // import "go.opentelemetry.io/collector/cmd/builder/internal/builder/modfile"

// This is the set of structs needed to interpret the output of `go
// mod edit -json` via the encoding/json unmarshaler.  This block of
// code is printed by `go help mod edit`, verbatim.

type Module struct {
	Path    string
	Version string
}

type GoMod struct {
	Module    ModPath
	Go        string
	Toolchain string
	Require   []Require
	Exclude   []Module
	Replace   []Replace
	Retract   []Retract
}

type ModPath struct {
	Path       string
	Deprecated string
}

type Require struct {
	Path     string
	Version  string
	Indirect bool
}

type Replace struct {
	Old Module
	New Module
}

type Retract struct {
	Low       string
	High      string
	Rationale string
}
