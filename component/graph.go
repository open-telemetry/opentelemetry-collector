// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

// Graph is a struct representing the pipelines
// currently registered with the `Host`.
type Graph struct {
	Pipelines []struct {
		FullName    string
		InputType   string
		MutatesData bool
		Receivers   []string
		Processors  []string
		Exporters   []string
	}
}
