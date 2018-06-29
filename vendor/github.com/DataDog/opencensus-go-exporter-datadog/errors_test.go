// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const waitTime = 10 * time.Millisecond

func containsFunc(t *testing.T) func(a error, b string) {
	return func(a error, b string) {
		if !strings.Contains(a.Error(), b) {
			t.Fatalf("%q did not contain %q", a.Error(), b)
		}
	}
}

func TestErrorAmortizer(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	t.Run("same", func(t *testing.T) {
		ma := newTestErrorAmortizer()
		for i := 0; i < 10; i++ {
			ma.log(errorTypeOverflow, errors.New("buffer full"))
		}
		time.Sleep(waitTime + 10*time.Millisecond)
		out := ma.lastError()
		if out == nil {
			t.Fatal("no error")
		}
		contains := containsFunc(t)
		contains(out, "Datadog Exporter error:")
		contains(out, "(x10)")
		contains(out, "buffer full")
	})

	t.Run("contention", func(t *testing.T) {
		ma := newTestErrorAmortizer()
		for i := 0; i < defaultErrorLimit+10; i++ {
			ma.log(errorTypeOverflow, nil)
		}
		time.Sleep(waitTime + 10*time.Millisecond)
		out := ma.lastError()
		if out == nil {
			t.Fatal("no error")
		}
		containsFunc(t)(out, fmt.Sprintf("Datadog Exporter error: span buffer overflow (x%d+)", defaultErrorLimit))
	})

	t.Run("various", func(t *testing.T) {
		ma := newTestErrorAmortizer()
		for j := 0; j < 2; j++ {
			ma.reset()
			for i := 0; i < 2; i++ {
				ma.log(errorTypeOverflow, nil)
			}
			for i := 0; i < 5; i++ {
				ma.log(errorTypeTransport, errors.New("transport failed"))
			}
			for i := 0; i < 3; i++ {
				ma.log(errorTypeEncoding, errors.New("encoding error"))
			}
			ma.log(errorTypeUnknown, errors.New("unknown error"))
			time.Sleep(waitTime + 10*time.Millisecond)
			out := ma.lastError()
			if out == nil {
				t.Fatal("no error")
			}
			contains := containsFunc(t)
			contains(out, "Datadog Exporter error:")
			contains(out, "span buffer overflow (x2)")
			contains(out, "transport failed (x5)")
			contains(out, "encoding error (x3)")
			contains(out, "unknown error")
		}
	})

	t.Run("one", func(t *testing.T) {
		ma := newTestErrorAmortizer()
		ma.log(errorTypeUnknown, errors.New("some error"))
		time.Sleep(waitTime + 10*time.Millisecond)
		out := ma.lastError()
		if out == nil {
			t.Fatal("no error")
		}
		contains := containsFunc(t)
		contains(out, "Datadog Exporter error:")
		contains(out, "some error")
	})
}

type testErrorAmortizer struct {
	*errorAmortizer

	mu      sync.RWMutex // guards lastErr
	lastErr error
}

func (ma *testErrorAmortizer) lastError() error {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	return ma.lastErr
}

func (ma *testErrorAmortizer) reset() {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	ma.lastErr = nil
}

func (ma *testErrorAmortizer) captureError(err error) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.lastErr = err
}

func newTestErrorAmortizer() *testErrorAmortizer {
	ea := newErrorAmortizer(waitTime, nil)
	ma := &testErrorAmortizer{errorAmortizer: ea}
	ma.errorAmortizer.callback = ma.captureError
	return ma
}
