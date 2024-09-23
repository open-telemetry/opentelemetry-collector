// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalgates // import "go.opentelemetry.io/collector/internal/globalgates"

import (
	"errors"

	"go.opentelemetry.io/collector/featuregate"
)

var NoopTracerProvider = featuregate.GlobalRegistry().MustRegister("service.noopTracerProvider",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.107.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("Sets a Noop OpenTelemetry TracerProvider to reduce memory allocations. This featuregate is incompatible with the zPages extension."))

// UseLocalHostAsDefaultHostfeatureGate is the feature gate that controls whether
// server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints.
var _ = mustRegisterOrLoad(
	featuregate.GlobalRegistry(),
	"component.UseLocalHostAsDefaultHost",
	featuregate.StageStable,
	featuregate.WithRegisterToVersion("v0.110.0"),
	featuregate.WithRegisterDescription("controls whether server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints"),
)

// mustRegisterOrLoad tries to register the feature gate and loads it if it already exists.
// It panics on any other error.
func mustRegisterOrLoad(reg *featuregate.Registry, id string, stage featuregate.Stage, opts ...featuregate.RegisterOption) *featuregate.Gate {
	gate, err := reg.Register(id, stage, opts...)

	if errors.Is(err, featuregate.ErrAlreadyRegistered) {
		// Gate is already registered; find it.
		// Only a handful of feature gates are registered, so it's fine to iterate over all of them.
		reg.VisitAll(func(g *featuregate.Gate) {
			if g.ID() == id {
				gate = g
				return
			}
		})
	} else if err != nil {
		panic(err)
	}

	return gate
}
