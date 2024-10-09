// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
)

func consume[T any](q Queue[T], consumeFunc func(context.Context, T) error) bool {
	index, req, ok := q.Read(context.Background())
	if !ok {
		return false
	}
	consumeErr := consumeFunc(context.Background(), req)
	q.OnProcessingFinished(index, consumeErr)
	return true
}
