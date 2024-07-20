// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/internal"
)

type ReceiverCreator interface {
	GetReceiverFactory(componentType component.Type) component.Factory
}

var NewReceiverCreator = internal.NewReceiverCreator
