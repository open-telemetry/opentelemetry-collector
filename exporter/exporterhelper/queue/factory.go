// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	intrequest "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Settings defines parameters for creating a Queue.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Settings struct {
	exporter.CreateSettings
	DataType component.DataType
	Callback func(req *intrequest.Request)
}

// Factory defines a factory interface for creating a Queue.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Factory interface {
	Create(Settings) Queue
}
