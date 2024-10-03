//go:build go1.23

package batchprocessor

import (
	"context"
	"testing"
	"time"
)

func TestShardTimer(t *testing.T) {
	_timer := time.NewTimer(1 * time.Millisecond)
	testShard := shard{
		processor: nil,
		exportCtx: context.Background(),
		timer:     _timer,
		newItem:   make(chan any),
		batch:     nil,
	}
	done := make(chan bool)
	go func() {
		time.Sleep(3 * time.Millisecond)
		testShard.stopTimer()
		done <- true
	}()
	select {
	case <-time.NewTimer(50 * time.Millisecond).C:
		t.Fail()
	case <-done:
		break
	}
}
