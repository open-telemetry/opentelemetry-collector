// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper instead.
package processorhelperprofiles // import "go.opentelemetry.io/collector/processor/processorhelper/processorhelperprofiles"

import "go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"

// Option apply changes to internalOptions.
// Deprecated: [0.116.0] Use xprocessorhelper.Option instead.
type Option = xprocessorhelper.Option

// WithStart overrides the default Start function for an processor.
// The default shutdown function does nothing and always returns nil.
// Deprecated: [0.116.0] Use xprocessorhelper.WithStart instead.
var WithStart = xprocessorhelper.WithStart

// WithShutdown overrides the default Shutdown function for an processor.
// The default shutdown function does nothing and always returns nil.
// Deprecated: [0.116.0] Use xprocessorhelper.WithShutdown instead.
var WithShutdown = xprocessorhelper.WithShutdown

// WithCapabilities overrides the default GetCapabilities function for an processor.
// The default GetCapabilities function returns mutable capabilities.
// Deprecated: [0.116.0] Use xprocessorhelper.WithCapabilities instead.
var WithCapabilities = xprocessorhelper.WithCapabilities
