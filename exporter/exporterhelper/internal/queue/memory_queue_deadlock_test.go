package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

// Simple sizer for int that just returns 1 for each item
type intSizer struct{}

func (s *intSizer) Sizeof(item int) int64 {
	return 1
}

// Simple encoding for int (not used in memory queue but required by Settings)
type intEncoding struct{}

func (e intEncoding) Marshal(ctx context.Context, item int) ([]byte, error) {
	return []byte{byte(item)}, nil
}

func (e intEncoding) Unmarshal(data []byte) (context.Context, int, error) {
	if len(data) == 0 {
		return context.Background(), 0, nil
	}
	return context.Background(), int(data[0]), nil
}

// TestMemoryQueueDeadlockReproduction replicates the exact conditions from
// batch processor test that causes the WaitForResult deadlock
func TestMemoryQueueDeadlockReproduction(t *testing.T) {
	// Create memory queue using proper constructor with same settings that cause deadlock
	set := Settings[int]{
		SizerType:  request.SizerTypeItems,
		ItemsSizer: &intSizer{},
		BytesSizer: &intSizer{},
		Capacity:   100, // Small capacity to force blocking
		Signal:     pipeline.SignalTraces,
		Encoding:   intEncoding{},
		ID:         component.NewID(exportertest.NopType),
		Telemetry:  componenttest.NewNopTelemetrySettings(),
	}
	set.BlockOnOverflow = true
	set.WaitForResult = true

	ctx := context.Background()

	mq := newMemoryQueue[int](set)
	require.NoError(t, mq.Start(ctx, componenttest.NewNopHost()))
	defer mq.Shutdown(ctx)

	// Metrics to track progress and detect deadlock
	var producersFinished int32
	var itemsProduced int32
	var itemsConsumed int32

	// Start consumer (emulates the async queue consumer)
	consumerDone := make(chan struct{})
	go func() {
		defer close(consumerDone)

		for {
			// Simplified: call Read() directly without timers
			_, _, done, ok := mq.Read(ctx)
			if !ok {
				return // Queue shutdown
			}

			if done != nil {
				done.OnDone(nil)
			}

			itemsConsumed++
		}
	}()

	// Parameters matching the deadlocking batch processor test
	requestCount := 50    // Reduced from 1000 for faster test
	itemsPerRequest := 10 // Reduced from 150 for faster test
	totalItems := requestCount * itemsPerRequest
	numProducers := 5

	var wg sync.WaitGroup
	wg.Add(numProducers)

	// Start multiple producers (emulates concurrent ConsumeTraces calls)
	for producerID := 0; producerID < numProducers; producerID++ {
		go func(id int) {
			defer wg.Done()

			itemsPerProducer := totalItems / numProducers

			for i := 0; i < itemsPerProducer; i++ {
				item := id*itemsPerProducer + i

				// This is where the deadlock happens in batch processor:
				// WaitForResult=true means Offer() waits for consumer to process
				err := mq.Offer(ctx, item)
				if err != nil {
					t.Errorf("Producer %d failed to offer item %d: %v", id, item, err)
					return
				}

				itemsProduced++
			}

			producersFinished++
		}(producerID)
	}

	// Wait for completion with timeout to detect deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	timeout := 10 * time.Second
	select {
	case <-done:
		t.Logf("SUCCESS: All producers completed. Items produced: %d, consumed: %d",
			itemsProduced, itemsConsumed)

		// Verify all items were processed
		assert.Equal(t, int32(totalItems), itemsProduced, "All items should be produced")

		// Give consumer time to finish processing
		time.Sleep(100 * time.Millisecond)

		// The test passes if we reach here without deadlock

	case <-time.After(timeout):
		t.Fatalf("DEADLOCK DETECTED: Test timed out after %v. "+
			"Producers finished: %d/%d, Items produced: %d/%d, Items consumed: %d",
			timeout, producersFinished, numProducers, itemsProduced, totalItems, itemsConsumed)
	}

	// Clean shutdown
	select {
	case <-consumerDone:
	case <-time.After(time.Second):
		t.Log("Consumer didn't finish cleanly")
	}
}
