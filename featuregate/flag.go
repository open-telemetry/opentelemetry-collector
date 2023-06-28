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
func NewFlag(reg *Registry) flag.Value {
	return &flagValue{reg: reg}
}

// flagValue implements the flag.Value interface and directly applies feature gate statuses to a Registry.
type flagValue struct {
	reg *Registry
}

func (f *flagValue) String() string {
	var ids []string
	if f.reg.strict {
		ids = []string{"strict"}
	}
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
	if len(ids) > 0 && ids[0] == "strict" {
		f.reg.strict = true
		ids = ids[1:]
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
		if _, ok := f.reg.gates.Load(id); !ok {
			errs = multierr.Append(errs, fmt.Errorf("no such feature gate %q", id))
		}
	}
	f.reg.VisitAll(func(gate *Gate) {
		enabled, ok := gatesEnabled[gate.id]
		if !ok {
			if f.reg.strict && (gate.stage == StageAlpha || gate.stage == StageBeta) {
				errs = multierr.Append(errs, fmt.Errorf("gate %q is not explicitly set", gate.id))
			}
			return
		}
		if f.reg.strict && !enabled && gate.stage == StageBeta {
			errs = multierr.Append(errs, fmt.Errorf("gate %s must be explicitly enabled, remove strict mode to override", gate.id))
		}
		if f.reg.strict && gate.stage == StageStable {
			errs = multierr.Append(errs, fmt.Errorf("gate %s must not be set", gate.id))
		}

		errs = multierr.Append(errs, f.reg.Set(gate.id, enabled))
	})

	return errs
}
