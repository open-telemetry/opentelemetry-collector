// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package storage implements an extension that can
// persist state beyond the collector process.
package storage // import "go.opentelemetry.io/collector/extension/experimental/storage"

import (
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

// Deprecated: [v0.117.0] use storage.NewNopClient from xextension.
var NewNopClient = storage.NewNopClient

// Deprecated: [v0.117.0] use storage.Client from xextension.
type Client = storage.Client

// Deprecated: [v0.117.0] use storage.Extension from xextension.
type Extension = storage.Extension

// Deprecated: [v0.117.0] use *storage.Operation from xextension.
type Operation = *storage.Operation

// Deprecated: [v0.117.0] use storage.SetOperation from xextension.
var SetOperation = storage.SetOperation

// Deprecated: [v0.117.0] use storage.GetOperation from xextension.
var GetOperation = storage.GetOperation

// Deprecated: [v0.117.0] use storage.DeleteOperation from xextension.
var DeleteOperation = storage.DeleteOperation

const (
	// Deprecated: [v0.117.0] use storage.NewNopClient from xextension.
	Get storage.OpType = iota
	// Deprecated: [v0.117.0] use storage.NewNopClient from xextension.
	Set
	// Deprecated: [v0.117.0] use storage.NewNopClient from xextension.
	Delete
)
