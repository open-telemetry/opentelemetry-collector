// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memorylimiterextension // import "go.opentelemetry.io/collector/extension/memorylimiterextension"

import "go.opentelemetry.io/collector/internal/memorylimiter"

type Config struct {
	memorylimiter.Config `mapstructure:",squash"`
	ApplyToAllReceivers  bool `mapstructure:"apply_to_all_receivers"`
}
