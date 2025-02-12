// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package hostcapabilities provides interfaces that can be implemented by the host
// to provide additional capabilities.
package hostcapabilities // import "go.opentelemetry.io/collector/service/hostcapabilities"

import (
	"go.opentelemetry.io/collector/service/internal/moduleinfo"
)

// ModuleInfo is an interface that may be implemented by the host to provide
// information about modules that were used to build the host.
type ModuleInfo interface {
	// GetModuleInfos returns the module information for the host
	// i.e. Receivers, Processors, Exporters, Extensions, and Connectors
	GetModuleInfos() moduleinfo.ModuleInfos
}
