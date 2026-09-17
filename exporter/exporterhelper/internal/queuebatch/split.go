// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// splitOps holds the operations splitRequest needs from one signal. T is the pdata
// type the request carries, and S is that signal's sizer: plog.Logs with
// sizer.LogsSizer, and so on.
//
// Every field is filled with a plain function or a method expression rather than a
// closure, so building one costs no allocation on a path that runs per request.
type splitOps[T, S any] struct {
	// itemCount reports how many items data holds, counting log records, data points
	// or spans depending on the signal.
	itemCount func(T) int
	// extract moves as many items as fit within maxSize out of data and reports the
	// size it took with them. It yields no items when the next one does not fit on
	// its own.
	extract func(data T, maxSize int, sz S) (T, int)
	// removeFirst drops the first item in iteration order, reporting whether there
	// was one to drop.
	removeFirst func(T) bool
	// size recomputes the size of data from scratch.
	size func(S, T) int
	// newRequest wraps extracted items in a request for the same signal.
	newRequest func(T) request.Request
	// itemName names a single item for the oversized error, e.g. "log record".
	itemName string
	// itemsName names several items for the unsplittable error, e.g. "log records".
	itemsName string
}

// splitRequest splits req into requests no larger than maxSize, dropping any single
// item too large to fit in a batch on its own. sizes is req's own size cache, which
// this reads and writes for szt as items leave req.
//
// Logs, metrics and traces differ only in the operations they pass in, so the loop
// lives here rather than once per signal.
func splitRequest[T, S any](
	req request.Request,
	sizes *request.SizeCache,
	data T,
	maxSize int,
	sz S,
	szt request.SizerType,
	cachedSize func() int,
	ops splitOps[T, S],
) ([]request.Request, error) {
	var res []request.Request
	droppedItems := 0
	unsplittable := false
	for cachedSize() > maxSize {
		extracted, removedSize := ops.extract(data, maxSize, sz)
		if ops.itemCount(extracted) == 0 {
			// The next item does not fit into maxSize even on its own, so no batch can
			// ever hold it. Drop only that item and keep splitting the rest, otherwise
			// every remaining item is discarded along with it.
			if !ops.removeFirst(data) {
				// There is no item left to drop, yet the request is still over maxSize,
				// so its resource and scope overhead alone exceeds the limit. Stop
				// instead of looping forever, and report it below rather than reporting
				// success for a request that was never split.
				unsplittable = true
				break
			}
			droppedItems++
			sizes.Update(szt, ops.size(sz, data))
			continue
		}
		sizes.Update(szt, cachedSize()-removedSize)
		res = append(res, ops.newRequest(extracted))
	}
	if unsplittable {
		// removeFirst prunes the scopes and resources it empties even when it finds no
		// item to remove, so the cached size is stale by this point.
		sizes.Update(szt, ops.size(sz, data))
	}
	// Keep the remainder, unless splitting emptied it, in which case there is nothing
	// left to export.
	if (droppedItems == 0 && !unsplittable) || ops.itemCount(data) > 0 {
		res = append(res, req)
	}
	switch {
	case droppedItems > 0:
		return res, fmt.Errorf("one %s size is greater than max size, dropping items: %d", ops.itemName, droppedItems)
	case unsplittable:
		return res, fmt.Errorf("request size is greater than max size and has no %s left to drop", ops.itemsName)
	}
	return res, nil
}
