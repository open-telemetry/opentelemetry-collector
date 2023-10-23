// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate // import "go.opentelemetry.io/collector/featuregate"

import (
	"flag"
	"strings"

	"go.uber.org/multierr"
)

// Flags related to feature gates.
type Flags struct {
	// RegistrationFlag is the flag used to register feature gates.
	RegistrationFlag flag.Value
}

// NewFlag returns a flag.Value that directly applies feature gate statuses to a Registry.
// Deprecated: Use NewFlags instead.
func NewFlag(reg *Registry) flag.Value {
	return &flagValue{reg: reg}
}

// NewFlags returns Flags that control building the global Registry.
func NewFlags(reg *Registry) Flags {
	return Flags{
		RegistrationFlag: &flagValue{reg: reg},
	}
}

// flagValue implements the flag.Value interface and directly applies feature gate statuses to a Registry.
type flagValue struct {
	reg *Registry
}

func (f *flagValue) String() string {
	var ids []string
	f.reg.VisitAll(func(g *Gate) {
		id := g.ID()
		if !g.IsEnabled() {
			id = "-" + id
		}
		ids = append(ids, id)
	})
	return strings.Join(ids, ",")
}

func (f *flagValue) Set(s string) error {
	if s == "" {
		return nil
	}

	var errs error
	ids := strings.Split(s, ",")
	for i := range ids {
		id := ids[i]
		val := true
		switch id[0] {
		case '-':
			id = id[1:]
			val = false
		case '+':
			id = id[1:]
		}
		errs = multierr.Append(errs, f.reg.Set(id, val))
	}
	return errs
}
