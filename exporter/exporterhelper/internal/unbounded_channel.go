// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

// newUnboundedChannel creates a pair of buffered channels with no cap.
// Caller should close the input channel when done. That will trigger draining of the output channel.
// Closing the output channel by client will panic.
func newUnboundedChannel[T any]() (chan<- T, <-chan T) {
	in, out := make(chan T), make(chan T)
	go func() {
		var queue []T
		for len(queue) > 0 || in != nil {
			var outCh chan T
			var curVal T
			if len(queue) > 0 {
				outCh = out
				curVal = queue[0]
			}
			select {
			case v, ok := <-in:
				if !ok {
					in = nil
				} else {
					queue = append(queue, v)
				}
			case outCh <- curVal:
				queue = queue[1:]
			}
		}
		close(out)
	}()
	return in, out
}
