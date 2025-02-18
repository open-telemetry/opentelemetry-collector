// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package moduleinfo // import "go.opentelemetry.io/collector/service/internal/moduleinfo"

import "go.opentelemetry.io/collector/component"

type ModuleInfo struct {
	// BuilderRef is the raw string passed in the builder configuration used to build this service.
	BuilderRef string
}

// ModuleInfos describes the go module for each component.
type ModuleInfos struct {
	Receiver  map[component.Type]ModuleInfo
	Processor map[component.Type]ModuleInfo
	Exporter  map[component.Type]ModuleInfo
	Extension map[component.Type]ModuleInfo
	Connector map[component.Type]ModuleInfo
}
