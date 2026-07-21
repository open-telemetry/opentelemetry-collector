// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package configstorage implements the configuration settings to
// communicate with the storage extension.
package configstorage // import "go.opentelemetry.io/collector/config/configstorage"

//go:generate mdatagen metadata.yaml

import (
	"context"
	"encoding"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

var (
	errNoStorageClient    = errors.New("no storage client extension found")
	errWrongExtensionType = errors.New("requested extension is not a storage extension")
)

// ID represents the ID of a storage extension component.
type ID component.ID

var (
	_ encoding.TextMarshaler   = ID{}
	_ encoding.TextUnmarshaler = (*ID)(nil)
)

// ComponentID returns the underlying component.ID.
func (id ID) ComponentID() component.ID {
	return component.ID(id)
}

// GetStorageClient returns a storage.Client from the extension identified by this ID.
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

// MarshalText implements the encoding.TextMarshaler interface.
func (id ID) MarshalText() ([]byte, error) {
	return component.ID(id).MarshalText()
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (id *ID) UnmarshalText(text []byte) error {
	return (*component.ID)(id).UnmarshalText(text)
}
