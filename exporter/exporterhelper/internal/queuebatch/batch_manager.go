// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

type batch struct {
	ctx     context.Context
	req     request.Request
	done    multiDone
	created time.Time
}

type shard struct {
	sync.Mutex
	*batch
}

func newShard() *shard {
	return &shard{
		sync.Mutex{},
		nil,
	}
}

type partitionManager interface {
	// getShard() returns a shard for the given request.
	getShard(ctx context.Context, req request.Request) *shard

	// forEachShard() iterates over all shards and calls the given callback function.
	forEachShard(func(*shard))
}

func newPartitionManager(partitioner Partitioner[request.Request]) partitionManager {
	if partitioner == nil {
		return newSinglePartitionManager()
	}
	return newMultiPartitionManager(partitioner)
}

type singlePartitionManager struct {
	shard *shard
}

func newSinglePartitionManager() *singlePartitionManager {
	return &singlePartitionManager{
		shard: newShard(),
	}
}

func (bm *singlePartitionManager) getShard(_ context.Context, _ request.Request) *shard {
	return bm.shard
}

func (bm *singlePartitionManager) forEachShard(callback func(*shard)) {
	callback(bm.shard)
}

type multiPartitionManager struct {
	// TODO: Currently, each partition can has only one shard. We need to support multiple shards per partition.
	partitionMap map[string]*shard
	partitioner  Partitioner[request.Request]
	mu           sync.RWMutex
}

func newMultiPartitionManager(partitioner Partitioner[request.Request]) *multiPartitionManager {
	return &multiPartitionManager{
		partitionMap: make(map[string]*shard),
		partitioner:  partitioner,
	}
}

// getShard() retursn the shard for the given request. It creates a new shard if the partition is not found.
func (bm *multiPartitionManager) getShard(ctx context.Context, req request.Request) *shard {
	key := bm.partitioner.GetKey(ctx, req)

	bm.mu.RLock()
	if shard, ok := bm.partitionMap[key]; ok {
		return shard
	}
	bm.mu.RUnlock()

	bm.mu.Lock()
	defer bm.mu.Unlock()
	shard := newShard()
	bm.partitionMap[key] = shard
	return shard
}

func (bm *multiPartitionManager) forEachShard(callback func(*shard)) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	for _, shard := range bm.partitionMap {
		callback(shard)
	}
}
