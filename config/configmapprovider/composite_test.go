// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmapprovider

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

func TestComposite_Watchable_Update(t *testing.T) {
	// Create a composite provider from 2 subproviders.

	// This is our compose func. It just returns a predictable config that is different
	// every time so that we can verify that Get() method returns what we expect.
	composeCounter := 0
	composeFunc := func(orig *config.Map, add *config.Map) error {
		composeCounter++
		orig.Set("counter", composeCounter)
		return nil
	}

	subProvider1 := newMockWatchableProvider(t)
	subProvider2 := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider1, subProvider2)
	require.NotNil(t, composite)

	// Retrieve the initial config from composite.
	retrieved, err := composite.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	cfg := retrieved.Get()
	assert.NotNil(t, cfg)

	// composeFunc should be called twice, so "counter" should be equal to 2.
	assert.EqualValues(t, 2, cfg.Get("counter"))

	// Now we are going to loop through [WatchForUpdate(), signal from subprovider
	// that there is an update, Get()] sequence a few times. We want to verify that:
	// - WatchForUpdate() of composite provider returns as expected.
	// - Get() returns the expected composed config.
	// - WatchForUpdate() can be restarted after it fires.
	for i := 0; i < 10; i++ {
		var wg sync.WaitGroup
		wg.Add(1)
		go func(i int) {
			// This should block until SignalUpdateAvailable() is called.
			err := retrieved.(WatchableRetrieved).WatchForUpdate()
			assert.Nil(t, err)

			// We have an updated config, get it.
			cfg := retrieved.Get()
			assert.NotNil(t, cfg)

			// Did we get the expected composed config? This should match the
			// "counter" value populated by composeFunc above. The "counter"
			// increases by 2 on every run.
			assert.EqualValues(t, (i+2)*2, cfg.Get("counter"))

			wg.Done()
		}(i)

		// SignalUpdateAvailable() should terminate WatchForUpdate().
		subProvider1.SignalUpdateAvailable(nil)

		// Wait for WatchForUpdate() to be terminated.
		wg.Wait()
	}

	assert.EqualValues(t, 0, subProvider1.GetCloseCallCounter())
	assert.EqualValues(t, 0, subProvider2.GetCloseCallCounter())

	// Close the composite provider. This should result in closing of subproviders.
	composite.Close(context.Background())

	// Verify that Close() of the subproviders was also called.
	assert.EqualValues(t, 1, subProvider1.GetCloseCallCounter())
	assert.EqualValues(t, 1, subProvider2.GetCloseCallCounter())
}

func TestComposite_Watchable_Close(t *testing.T) {
	// Create a composite provider from 1 subprovider.
	composeFunc := func(orig *config.Map, add *config.Map) error {
		return nil
	}

	subProvider := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider)
	require.NotNil(t, composite)

	// Retrieve the initial config.
	retrieved, err := composite.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		cfg := retrieved.Get()
		assert.NotNil(t, cfg)

		// This should block until Close() is called.
		err := retrieved.(WatchableRetrieved).WatchForUpdate()

		// Verify that the correct error is returned as a result of Close().
		assert.ErrorIs(t, err, configsource.ErrSessionClosed)

		wg.Done()
	}()

	// Close the composite provider. Close() should terminate WatchForUpdate().
	composite.Close(context.Background())

	// Wait for WatchForUpdate() to be terminated.
	wg.Wait()

	// Verify that Close() of the subprovider was also called.
	assert.EqualValues(t, 1, subProvider.GetCloseCallCounter())
}

func TestComposite_Watchable_Subprovider_Terminate(t *testing.T) {
	// Create a composite provider from 2 subproviders.
	composeFunc := func(orig *config.Map, add *config.Map) error {
		return nil
	}

	subProvider := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider)
	require.NotNil(t, composite)

	// Retrieve the initial config.
	retrieved, err := composite.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		cfg := retrieved.Get()
		assert.NotNil(t, cfg)

		// This should block until subprovider's WatchForUpdate() is terminated.
		err := retrieved.(WatchableRetrieved).WatchForUpdate()

		// Verify that the correct error is returned as a result of subprovider's
		// termination.
		assert.ErrorIs(t, err, configsource.ErrSessionClosed)
		wg.Done()
	}()

	// SignalUpdateAvailable() will terminate the subprovider's WatchForUpdate()
	// permanently and make it return ErrSessionClosed.
	// This should propagate the termination to WatchForUpdate() of composite's
	// retrieved that we wait for in the go routine above.
	subProvider.SignalUpdateAvailable(configsource.ErrSessionClosed)

	wg.Wait()

	// Verify that Close() of the subprovider was not yet called.
	assert.EqualValues(t, 0, subProvider.GetCloseCallCounter())

	composite.Close(context.Background())

	// Verify that Close() of the subprovider was now called.
	assert.EqualValues(t, 1, subProvider.GetCloseCallCounter())
}

func TestComposite_Watchable_FailCompose(t *testing.T) {
	composeFunc := func(orig *config.Map, add *config.Map) error {
		return errors.New("cannot compose configs")
	}

	subProvider := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider)
	require.NotNil(t, composite)
	retrieved, err := composite.Retrieve(context.Background())
	assert.Error(t, err)
	assert.Nil(t, retrieved)

	// Verify that Close() of the subprovider was not yet called.
	assert.EqualValues(t, 0, subProvider.GetCloseCallCounter())

	composite.Close(context.Background())

	// Verify that Close() of the subprovider was now called.
	assert.EqualValues(t, 1, subProvider.GetCloseCallCounter())
}

func TestComposite_Watchable_FailCompose_AfterUpdate(t *testing.T) {
	var composeError error
	composeFunc := func(orig *config.Map, add *config.Map) error {
		return composeError
	}

	subProvider := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider)
	require.NotNil(t, composite)
	retrieved, err := composite.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		cfg := retrieved.Get()
		assert.NotNil(t, cfg)

		// This should block until subprovider's WatchForUpdate() is terminated.
		err := retrieved.(WatchableRetrieved).WatchForUpdate()

		// Verify that the correct error is returned. Become composeFunc failed, its
		// returned error should be propagated to WatchForUpdate().
		assert.ErrorIs(t, err, composeError)
		wg.Done()
	}()

	// Next composeFunc call should return an error.
	composeError = errors.New("cannot compose configs")

	// Signal WatchForUpdate() to return. This will begin a new retrieving session
	// and the composite will call Get() of subproviders and will then call composeFunc.
	// composeFunc should fail this time and the composite's WatchForUpdate() should
	// fail too and sohuld return an error.
	subProvider.SignalUpdateAvailable(nil)

	wg.Wait()
}

func TestComposite_Subprovider_Close_Fail(t *testing.T) {
	// Create a composite provider from 2 subproviders.
	composeFunc := func(orig *config.Map, add *config.Map) error {
		return nil
	}

	subProvider1 := newMockWatchableProvider(t)
	subProvider2 := newMockWatchableProvider(t)

	composite := NewComposite(composeFunc, subProvider1, subProvider2)
	require.NotNil(t, composite)

	// Retrieve the initial config from composite.
	retrieved, err := composite.Retrieve(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	subErr := errors.New("bad failure")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		cfg := retrieved.Get()
		assert.NotNil(t, cfg)

		// This should block until subprovider's WatchForUpdate() is terminated.
		err = retrieved.(WatchableRetrieved).WatchForUpdate()

		// Verify that the correct error is returned as a result of subprovider's
		// termination.
		assert.ErrorIs(t, err, subErr)
		wg.Done()
	}()

	// Prepare the first subprovider to return an error on close.
	subProvider1.SetCloseReturnError(subErr)

	// Signal that the second subprovider has a regular update. This should trigger
	// the call to Close() of the first subprovider which should return an error.
	// This in turn should result in WatchForUpdate() of the composite
	// to return that same error, which we check above in the go routine.
	subProvider2.SignalUpdateAvailable(nil)

	wg.Wait()

	// Close the composite provider. This should result in closing of subproviders.
	err = composite.Close(context.Background())

	// We should also get the same subprovider's error here.
	assert.ErrorIs(t, err, subErr)

	// Verify that Close() of the subproviders was called.
	assert.EqualValues(t, 1, subProvider1.GetCloseCallCounter())
	assert.EqualValues(t, 1, subProvider2.GetCloseCallCounter())
}

// A map provider which is watchable and can be forces to return from WatchForUpdate.
type mockWatchableProvider struct {
	watchSignal      chan error
	closeCallCounter uint64
	closeReturnError error
	retrieved        *mockWatchableRetrieved
	t                *testing.T
}

func newMockWatchableProvider(t *testing.T) *mockWatchableProvider {
	return &mockWatchableProvider{
		watchSignal: make(chan error, 1),
		t:           t,
	}
}

func (p *mockWatchableProvider) Retrieve(context.Context) (Retrieved, error) {
	p.retrieved = &mockWatchableRetrieved{watchSignal: p.watchSignal, t: p.t}
	return p.retrieved, nil
}

func (p *mockWatchableProvider) Close(context.Context) error {
	atomic.AddUint64(&p.closeCallCounter, 1)
	return p.closeReturnError
}

func (p *mockWatchableProvider) GetCloseCallCounter() uint64 {
	return atomic.LoadUint64(&p.closeCallCounter)
}

func (p *mockWatchableProvider) SetCloseReturnError(err error) {
	p.closeReturnError = err
	if p.retrieved != nil {
		p.retrieved.closeReturnError = err
	}
}

func (p *mockWatchableProvider) SignalUpdateAvailable(withErr error) {
	// Signal to WatchForUpdate to terminate and return.
	p.watchSignal <- withErr
}

type mockWatchableRetrieved struct {
	watchSignal           chan error
	closeReturnError      error
	watchForUpdateCounter int64
	t                     *testing.T
}

func (r mockWatchableRetrieved) Get() *config.Map {
	return config.NewMapFromStringMap(map[string]interface{}{})
}

func (r *mockWatchableRetrieved) WatchForUpdate() error {
	watchEntranceCounter := atomic.AddInt64(&r.watchForUpdateCounter, 1)
	defer atomic.AddInt64(&r.watchForUpdateCounter, -1)
	assert.EqualValues(r.t, 1, watchEntranceCounter)

	// Wait until signaled by SignalUpdateAvailable().
	err := <-r.watchSignal
	return err
}

func (r *mockWatchableRetrieved) Close(ctx context.Context) error {
	// Signal to WatchForUpdate to terminate and return.
	select {
	case r.watchSignal <- configsource.ErrSessionClosed:
	default:
	}
	return r.closeReturnError
}
