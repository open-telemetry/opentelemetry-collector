// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import (
	"go.opentelemetry.io/collector/component"
)

type ReceiverCreator struct {
	receiverFactoryFunc func(componentType component.Type) Factory
}

func (r ReceiverCreator) GetReceiverFactory(componentType component.Type) component.Factory {
	return r.receiverFactoryFunc(componentType)
}

func NewReceiverCreator(settings Settings) ReceiverCreator {
	return ReceiverCreator{
		receiverFactoryFunc: settings.receiverFactoryFunc,
	}
}
