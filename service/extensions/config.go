// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import "go.opentelemetry.io/collector/component"

// Config represents the ordered list of extensions configured for the service.
type Config []component.ID
