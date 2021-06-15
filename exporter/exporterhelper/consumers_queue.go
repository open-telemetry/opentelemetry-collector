// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper

// consumersQueue is largely based on queue.BoundedQueue and matches the subset used in the collector
// It describes a producer-consumer exchange which can be backed by e.g. the memory-based ring buffer queue
// (queue.BoundedQueue) or via disk-based queue (persistentQueue)
type consumersQueue interface {
	// StartConsumers starts a given number of goroutines consuming items from the queue
	// and passing them into the consumer callback.
	StartConsumers(num int, callback func(item interface{}))
	// Produce is used by the producer to submit new item to the queue. Returns false in case of queue overflow.
	Produce(item interface{}) bool
	// Stop stops all consumers, as well as the length reporter if started,
	// and releases the items channel. It blocks until all consumers have stopped.
	Stop()
	// Size returns the current Size of the queue
	Size() int
}
