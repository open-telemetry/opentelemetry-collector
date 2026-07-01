// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configstorage implements the configuration settings to
// communicate with the storage extension.
package configstorage // import "go.opentelemetry.io/collector/config/configstorage"

//go:generate mdatagen metadata.yaml

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

var (
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

type ID component.ID

func (id ID) ComponentID() component.ID {
	return component.ID(id)
}

func (id ID) GetStorageClient(ctx context.Context, kind component.Kind, host component.Host, ownerID component.ID, name string) (storage.Client, error) {
	ext, found := host.GetExtensions()[id.ComponentID()]
	if !found {
		return nil, errNoStorageClient
	}

	storageExt, ok := ext.(storage.Extension)
	if !ok {
		return nil, errWrongExtensionType
	}

	return storageExt.GetClient(ctx, kind, ownerID, name)
}

func (id ID) MarshalText() ([]byte, error) {
	return component.ID(id).MarshalText()
}

func (id *ID) UnmarshalText(text []byte) error {
	return (*component.ID)(id).UnmarshalText(text)
}
