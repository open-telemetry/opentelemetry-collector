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
func NewFlag(reg *Registry, strict bool) flag.Value {
	return &flagValue{reg: reg, strict: strict}
}

// flagValue implements the flag.Value interface and directly applies feature gate statuses to a Registry.
type flagValue struct {
	reg    *Registry
	strict bool
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
		if _, ok := f.reg.gates.Load(id); !ok {
			errs = multierr.Append(errs, fmt.Errorf("no such feature gate %q", id))
		}
	}
	f.reg.VisitAll(func(gate *Gate) {
		enabled, ok := gatesEnabled[gate.id]
		if !ok {
			if f.strict && (gate.stage == StageAlpha || gate.stage == StageBeta) {
				errs = multierr.Append(errs, fmt.Errorf("gate %q is not explicitly configured", gate.id))
			}
			return
		}
		if f.strict && !enabled && gate.stage == StageBeta {
			errs = multierr.Append(errs, fmt.Errorf("gate %q is in beta and must be explicitly enabled", gate.id))
		}
		if f.strict && gate.stage == StageStable {
			errs = multierr.Append(errs, fmt.Errorf("gate %q is stable and must not be configured", gate.id))
		}

		errs = multierr.Append(errs, f.reg.Set(gate.id, enabled))
	})

	return errs
}
