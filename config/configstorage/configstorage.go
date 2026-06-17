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

type Config struct {
	//  specifies the name of the storage extension to be used
	ID component.ID `mapstructure:"id,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

func (c *Config) Validate() error {
	if c.ID == (component.ID{}) {
		return errors.New("storage 'id' is required")
	}
	return nil
}

func (c Config) GetStorageClient(ctx context.Context, kind component.Kind, host component.Host, ownerID component.ID, name string) (storage.Client, error) {
	ext, found := host.GetExtensions()[c.ID]
	if !found {
		return nil, errNoStorageClient
	}

	storageExt, ok := ext.(storage.Extension)
	if !ok {
		return nil, errWrongExtensionType
	}

	return storageExt.GetClient(ctx, kind, ownerID, name)
}
