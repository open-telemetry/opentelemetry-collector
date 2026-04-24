// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package storage // import "go.opentelemetry.io/collector/extension/xextension/storage"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Extension is the interface that storage extensions must implement
type Extension interface {
	extension.Extension

	// GetClient will create a client for use by the specified component.
	// Each component can have multiple storages (e.g. one for each signal),
	// which can be identified using storageName parameter.
	// The component can use the client to manage state
	GetClient(ctx context.Context, kind component.Kind, id component.ID, storageName string) (Client, error)
}

// Client is the interface that storage clients must implement
// All methods should return error only if a problem occurred.
// This mirrors the behavior of a golang map:
//   - Set doesn't error if a key already exists - it just overwrites the value.
//   - Get doesn't error if a key is not found - it just returns nil.
//   - Delete doesn't error if the key doesn't exist - it just no-ops.
//
// Similarly:
//   - Batch doesn't error if any of the above happens for either retrieved or updated keys
//
// This also provides a way to differentiate data operations
//
//	[overwrite | not-found | no-op] from "real" problems
type Client interface {
	// Get will retrieve data from storage that corresponds to the
	// specified key. It should return (nil, nil) if not found
	Get(ctx context.Context, key string) ([]byte, error)

	// Set will store data. The data can be retrieved by the same
	// component after a process restart, using the same key
	Set(ctx context.Context, key string, value []byte) error

	// Delete will delete data associated with the specified key
	Delete(ctx context.Context, key string) error

	// Batch handles specified operations in batch. Get operation results are put in-place
	Batch(ctx context.Context, ops ...*Operation) error

	// Close will release any resources held by the client
	Close(ctx context.Context) error
}

// WalkFunc is the type of the function called by Walk to visit each entry in the storage.
//
// The key argument contains the key of the stored entry.
// Key order is not guaranteed and may vary between storage implementations.
//
// The value bytes are only valid for the duration of the
// function call and need to be copied if later access is needed.
//
// The function may return operations that are collected and applied in order
// either when the end of the Walk is reached or after SkipAll is returned.
// Returning a nil slice is valid and simply contributes no operations.
// All operation types (Get, Set, Delete) are valid in the returned slice.
// Get operations work as usual — the result is stored in the Operation instance.
//
// If the storage supports transactions, all returned operations must be
// applied in the same transaction the key/value entries for WalkFunc were obtained from.
//
// The error result returned by the function controls how Walk continues:
//
// If the function returns the special value SkipAll, Walk skips all remaining
// storage entries and all collected operations still get applied in order.
//
// Otherwise, if the function returns a non-nil error,
// Walk stops entirely and returns that error without applying
// any of the collected operations.
type WalkFunc func(key string, value []byte) ([]*Operation, error)

// SkipAll is used as a return value from WalkFunc to indicate that
// all remaining storage entries are to be skipped.
// The pending operations are still applied.
var SkipAll = errors.New("skip everything and stop the walk")

// Walker is the interface that a storage extension Client may implement
// in order to support ranging through the storage entries.
type Walker interface {
	// Walk calls fn for every key/value pair in the storage.
	//
	// If fn returns a non-nil error other than SkipAll, or if Walk
	// encounters an internal error, ranging stops immediately and no
	// collected operations are applied.
	Walk(ctx context.Context, fn WalkFunc) error
}

type OpType int

const (
	Get OpType = iota
	Set
	Delete
)

type Operation struct {
	// Key specifies key which is going to be get/set/deleted
	Key string
	// Value specifies value that is going to be set or holds result of get operation
	Value []byte
	// Type describes the operation type
	Type OpType
}

func SetOperation(key string, value []byte) *Operation {
	return &Operation{
		Key:   key,
		Value: value,
		Type:  Set,
	}
}

func GetOperation(key string) *Operation {
	return &Operation{
		Key:  key,
		Type: Get,
	}
}

func DeleteOperation(key string) *Operation {
	return &Operation{
		Key:  key,
		Type: Delete,
	}
}

var ErrStorageFull = errors.New("the storage extension has run out of available space")
