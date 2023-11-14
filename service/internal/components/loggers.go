// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components // import "go.opentelemetry.io/collector/service/internal/components"

import (
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
)

const (
	zapKindKey            = "kind"
	zapNameKey            = "name"
	zapDataTypeKey        = "data_type"
	zapStabilityKey       = "stability"
	zapPipelineKey        = "pipeline"
	zapExporterInPipeline = "exporter_in_pipeline"
	zapReceiverInPipeline = "receiver_in_pipeline"
)

func ReceiverLogger(logger *zap.Logger, id component.ID, dt component.DataType) *zap.Logger {
	return logger.With(
		zap.String(zapKindKey, strings.ToLower(component.KindReceiver.String())),
		zap.String(zapNameKey, id.String()),
		zap.String(zapDataTypeKey, string(dt)))
}

func ProcessorLogger(logger *zap.Logger, id component.ID, pipelineID component.ID) *zap.Logger {
	return logger.With(
		zap.String(zapKindKey, strings.ToLower(component.KindProcessor.String())),
		zap.String(zapNameKey, id.String()),
		zap.String(zapPipelineKey, pipelineID.String()))
}

func ExporterLogger(logger *zap.Logger, id component.ID, dt component.DataType) *zap.Logger {
	return logger.With(
		zap.String(zapKindKey, strings.ToLower(component.KindExporter.String())),
		zap.String(zapDataTypeKey, string(dt)),
		zap.String(zapNameKey, id.String()))
}

func ExtensionLogger(logger *zap.Logger, id component.ID) *zap.Logger {
	return logger.With(
		zap.String(zapKindKey, strings.ToLower(component.KindExtension.String())),
		zap.String(zapNameKey, id.String()))
}

func ConnectorLogger(logger *zap.Logger, id component.ID, expDT, rcvDT component.DataType) *zap.Logger {
	return logger.With(
		zap.String(zapKindKey, strings.ToLower(component.KindConnector.String())),
		zap.String(zapNameKey, id.String()),
		zap.String(zapExporterInPipeline, string(expDT)),
		zap.String(zapReceiverInPipeline, string(rcvDT)))
}
