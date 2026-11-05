// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"
)

// UpdateReceivers applies receiver-only changes using Update.
func (g *Graph) UpdateReceivers(ctx context.Context, set Settings,
	changedReceivers map[component.ID]bool,
	newReceiverCfgs map[component.ID]component.Config,
	receiverFactories map[component.Type]receiver.Factory,
	host *Host,
) error {
	_, err := g.Update(ctx, set, UpdateRequest{
		ChangedReceivers:  changedReceivers,
		NewReceiverCfgs:   newReceiverCfgs,
		ReceiverFactories: receiverFactories,
	}, host)
	return err
}
