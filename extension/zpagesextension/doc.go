// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate env GITHUB_PROJECT=open-telemetry/opentelemetry-collector mdatagen metadata.yaml

// Package zpagesextension implements an extension that exposes zPages of
// properly instrumented components.
package zpagesextension // import "go.opentelemetry.io/collector/extension/zpagesextension"
