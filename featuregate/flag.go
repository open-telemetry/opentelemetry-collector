// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate // import "go.opentelemetry.io/collector/featuregate"

import (
	"flag"
	"fmt"
	"strings"

	"go.uber.org/multierr"
)

// NewFlag returns a flag.Value that directly applies feature gate statuses to a Registry.
//
// Deprecated: use NewFlags instead to get both gates and strict flags.
func NewFlag(reg *Registry) flag.Value {
	return &gatesFlagValue{args: &gateRegistrationArgs{reg: reg}}
}

// NewFlags returns two flag.Values: one that registers gates, and one that sets strict mode.
func NewFlags(reg *Registry) (flag.Value, flag.Value) {
	g := &gateRegistrationArgs{reg: reg}
	return &gatesFlagValue{args: g}, &strictFlagValue{args: g}
}

// gatesFlagValue implements the flag.Value interface and records gates to set on the Registry.
type gatesFlagValue struct {
	args *gateRegistrationArgs
}

func (f *gatesFlagValue) String() string {
	var gates []string
	f.args.reg.VisitAll(func(gate *Gate) {
		id := gate.ID()
		if !gate.IsEnabled() {
			id = "-" + id
		}
		gates = append(gates, id)
	})
	return strings.Join(gates, ",")
}

func (f *gatesFlagValue) Set(s string) error {
	f.args.gates = s
	f.args.gatesSet = true
	return f.args.set()
}

// strictFlagValue implements the flag.Value interface and sets strict mode on the validation.
type strictFlagValue struct {
	args *gateRegistrationArgs
}

func (f *strictFlagValue) IsBoolFlag() bool {
	return true
}

func (f *strictFlagValue) String() string {
	return fmt.Sprintf("%t", f.args.strict)
}

func (f *strictFlagValue) Set(s string) error {
	if s == "true" {
		f.args.strict = true
	}
	f.args.strictSet = true
	return f.args.set()
}

type gateRegistrationArgs struct {
	reg       *Registry
	gates     string
	strict    bool
	strictSet bool
	gatesSet  bool
}

func (g *gateRegistrationArgs) set() error {
	if !g.strictSet || !g.gatesSet {
		return nil
	}
	var errs error
	var ids []string
	if g.gates != "" {
		ids = strings.Split(g.gates, ",")
	}
	gatesEnabled := map[string]bool{}

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
		gatesEnabled[id] = val
		if _, ok := g.reg.gates.Load(id); !ok {
			errs = multierr.Append(errs, fmt.Errorf("no such feature gate %q", id))
		}
	}
	g.reg.VisitAll(func(gate *Gate) {
		enabled, ok := gatesEnabled[gate.id]
		if !ok {
			if g.strict && (gate.stage == StageAlpha || gate.stage == StageBeta) {
				errs = multierr.Append(errs, fmt.Errorf("gate %q is in %s and must be explicitly configured", gate.id, gate.stage))
			}
			return
		}
		if g.strict && !enabled && gate.stage == StageBeta {
			errs = multierr.Append(errs, fmt.Errorf("gate %q is in beta and must be explicitly enabled", gate.id))
		}
		if g.strict && gate.stage == StageStable {
			errs = multierr.Append(errs, fmt.Errorf("gate %q is stable and must not be configured", gate.id))
		}

		errs = multierr.Append(errs, g.reg.Set(gate.id, enabled))
	})

	return errs
}
