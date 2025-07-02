// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package featuregate provides mechanisms for feature flagging in the OpenTelemetry Collector.
//
// Example (Checking if a feature gate is enabled):
//
//	gate := featuregate.GlobalRegistry().MustRegister(
//		"example.feature",
//		featuregate.StageAlpha,
//	)
//
//	if gate.IsEnabled() {
//		// Enable new feature behavior
//	}
//
// Example (Registering, setting, and visiting feature gates):
//
//	reg := featuregate.NewRegistry()
//	gate := reg.MustRegister("example.feature2", featuregate.StageAlpha)
//	_ = reg.Set(gate.ID(), true) // Enable the gate
//	reg.VisitAll(func(g *featuregate.Gate) {
//		fmt.Printf("Gate: %s, Enabled: %v\n", g.ID(), g.IsEnabled())
//	})
package featuregate
