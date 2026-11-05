// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package refconsumer // import "go.opentelemetry.io/collector/service/internal/refconsumer"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
)

func NewProfiles(cons xconsumer.Profiles) xconsumer.Profiles {
	return refProfiles{
		consumer: cons,
	}
}

type refProfiles struct {
	consumer xconsumer.Profiles
}

// ConsumeProfiles takes pipeline ownership of the data and releases it once the downstream consumer
// returns. It is a no-op if the data is already owned by the pipeline.
func (c refProfiles) ConsumeProfiles(ctx context.Context, ld pprofile.Profiles) error {
	if pref.MarkPipelineOwnedProfiles(ld) {
		defer pref.UnrefProfiles(ld)
	}
	return c.consumer.ConsumeProfiles(ctx, ld)
}

func (c refProfiles) Capabilities() consumer.Capabilities {
	return c.consumer.Capabilities()
}
