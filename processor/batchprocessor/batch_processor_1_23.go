// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build go1.23

package batchprocessor // import "go.opentelemetry.io/collector/processor/batchprocessor"

func (b *shard) stopTimer() {
	if b.hasTimer() {
		b.timer.Stop()
	}
}
