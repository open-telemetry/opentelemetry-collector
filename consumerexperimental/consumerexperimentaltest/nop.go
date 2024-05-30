// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumerexperimentaltest // import "go.opentelemetry.io/collector/consumer/consumerexperimentaltest"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

// NewNop returns a Consumer that just drops all received data and returns no error.
func NewNop() Consumer {
	return &baseConsumer{
		ConsumeProfilesFunc: func(context.Context, pprofile.Profiles) error { return nil },
	}
}
