// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package moduleinfo // import "go.opentelemetry.io/collector/service/internal/moduleinfo"

import "go.opentelemetry.io/collector/component"

// ModuleInfo describes the go module for each component.
type ModuleInfo struct {
	Receiver  map[component.Type]string
	Processor map[component.Type]string
	Exporter  map[component.Type]string
	Extension map[component.Type]string
	Connector map[component.Type]string
}
