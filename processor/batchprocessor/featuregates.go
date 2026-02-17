// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

import (
	"go.opentelemetry.io/collector/featuregate"
)

const (
	// useExporterHelperGate controls whether to use exporterhelper components
	// for batching instead of the legacy implementation.
	useExporterHelperGate = "processor.batch.useExporterHelper"

	// propagateErrorsGate controls whether to propagate errors from the next
	// consumer instead of suppressing them (legacy behavior).
	propagateErrorsGate = "processor.batch.propagateErrors"
)

var (
	// UseExporterHelper is the feature gate for using exporterhelper components.
	useExporterHelper = featuregate.GlobalRegistry().MustRegister(
		useExporterHelperGate,
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Use exporterhelper components for batching instead of legacy implementation"),
		featuregate.WithRegisterFromVersion("v0.131.0"),
	)

	// PropagateErrors is the feature gate for propagating errors instead of suppressing them.
	propagateErrors = featuregate.GlobalRegistry().MustRegister(
		propagateErrorsGate,
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("Propagate errors from next consumer instead of suppressing them"),
		featuregate.WithRegisterFromVersion("v0.131.0"),
	)
)
