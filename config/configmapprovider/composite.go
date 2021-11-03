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

package configmapprovider // import "go.opentelemetry.io/collector/config/configmapprovider"

import (
	"context"

	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

// configComposeFunc combines a current config "orig" with another config "add".
// The resulting config is stored in "orig".
type configComposeFunc func(orig *config.Map, add *config.Map) error

// compositeMapProvider creates a map provider that is a composite of other map providers.
// The composition function that combines the config maps returned by the providers
// must be provided externally.
type compositeMapProvider struct {
	composeFunc  configComposeFunc
	subProviders []Provider
	retrieved    *compositeRetrieved
}

// NewComposite returns a Provider, that combines multiple
// Providers.
//
// The resulting config.Map is created by composing the config.Maps returned
// by each subprovider in the specified order into an initially empty config.Map
// using the composeFunc.
//
// The returned map provider's Retrieved object also implements a WatchableRetrieved
// interface which is triggered when a WatchableRetrieved of any subprovider is triggered.
//
// TODO: pass logger to compositeMapProvider for debug logging and to pass to
// subproviders such as fileMapProvider to print when the file is changed.
func NewComposite(composeFunc configComposeFunc, subProviders ...Provider) Provider {
	return &compositeMapProvider{composeFunc: composeFunc, subProviders: subProviders}
}

// A list of Retrieved objects.
type retrievedList []Retrieved

// getAllAndCompose calls Get() of each subprovider, then uses the composeFunc to combine
// each returned config.Map into one config.Map and return the resulting map.
func (rl retrievedList) getAllAndCompose(composeFunc configComposeFunc) (*config.Map, error) {
	// We start with an empty map.
	retCfgMap := config.NewMap()
	for _, retrieved := range rl {
		// Get a map from subprovider.
		configMap := retrieved.Get()

		// And compose it on top of the current map.
		if err := composeFunc(retCfgMap, configMap); err != nil {
			return nil, err
		}
	}

	return retCfgMap, nil
}

func (mp *compositeMapProvider) Retrieve(ctx context.Context) (Retrieved, error) {
	rList := retrievedList{}

	// Call Retrieve() of every subprovider and remember the resulting Retrieved.
	for _, p := range mp.subProviders {
		retr, err := p.Retrieve(ctx)
		if err != nil {
			return nil, err
		}
		rList = append(rList, retr)
	}

	// Create a Retrieved object that has the list of subprovider's Retrieved objects.
	var err error
	mp.retrieved, err = newCompositeRetrieved(ctx, mp.composeFunc, rList)
	return mp.retrieved, err
}

func (mp *compositeMapProvider) Close(ctx context.Context) error {
	var errs error

	// If Retrieved() was called close it.
	if mp.retrieved != nil {
		err := mp.retrieved.Close(ctx)
		errs = multierr.Append(errs, err)
	}

	// Close all subproviders.
	for _, sub := range mp.subProviders {
		err := sub.Close(ctx)
		errs = multierr.Append(errs, err)
	}

	return errs
}

// compositeRetrieved tracks a list of subproviders' Retrieved objects and watches
// for WatchForUpdate() of these Retrieved to return.
// ┌────────────────────┐
// │ compositeRetrieved │
// └─────────┬──────────┘
//           │
//           │   ┌─────────┬─────────┬─────────┬───┐
//           └──►│Retrieved│Retrieved│Retrieved│...│
//               └─────────┴─────────┴─────────┴───┘
type compositeRetrieved struct {
	// The current config map composed from subprovider's config maps. Will be returned
	// by Get() func.
	composedMap *config.Map

	composeFunc configComposeFunc

	// List of retrieved subproviders.
	subRetrieved retrievedList

	// A cancellable context that is used for long-running operations in WatchForUpdate().
	watchContext       context.Context
	cancelWatchContext func()
}

// A Watchable of which WaitForUpdate() returned and that is ready for Get() call.
type readyWatchable struct {
	// The Watchable that is ready.
	watchable WatchableRetrieved

	// The value that was returned by WaitForUpdate().
	returnedError error
}

// Create a new composite Retrieved object from a list of Retrieved objects returned
// by subproviders' Retrieve() calls.
func newCompositeRetrieved(
	ctx context.Context,
	composeFunc configComposeFunc,
	subRetrieved retrievedList,
) (*compositeRetrieved, error) {
	// Unfortunately Retrieved.Get() cannot return an error, but it is
	// possible that composeFunc returns an error. Because of that we cannot
	// do the composition using composeFunc in our Get() implementation and
	// have to do it here so that we can return an error.
	// TODO: change Retrieved.Get() to allow returning an error and move
	// the following code to our Get().

	// Call Get() on all sub-retrieved and combine the results using the composeFunc.
	composedMap, err := subRetrieved.getAllAndCompose(composeFunc)
	if err != nil {
		// Didn't work. Close every WatchableRetrieved that may be on our list.
		for _, retr := range subRetrieved {
			if watchable, ok := retr.(WatchableRetrieved); ok {
				err2 := watchable.Close(ctx)
				err = multierr.Append(err, err2)
			}
		}
		return nil, err
	}

	// Create a new cancellable context for long-running operations.
	ctx, cancelFunc := context.WithCancel(context.Background())

	m := &compositeRetrieved{
		composedMap:        composedMap,
		composeFunc:        composeFunc,
		subRetrieved:       subRetrieved,
		watchContext:       ctx,
		cancelWatchContext: cancelFunc,
	}

	return m, nil
}

func (cr *compositeRetrieved) Get() *config.Map {
	// WatchableRetrieved requires that Get() is never called concurrently
	// with WatchForUpdate() which means a race on cr.composedMap is not possible
	// since the only place where we modify it is in WatchForUpdate(), so there
	// is no need to protect cr.composedMap from races.
	return cr.composedMap
}

func (cr *compositeRetrieved) WatchForUpdate() error {
	// Create one go routine per sub-retrieved and start waiting for WatchForUpdate()
	// of any Watchable sub-retrieved to return.

	// Channel used to signal that sub-retrieved WaitForUpdate() returned.
	readyWatchableCh := make(chan readyWatchable)

	// Count how many of the sub-retrieved are watchable.
	watchableCount := 0
	for _, subRetrieved := range cr.subRetrieved {
		if watchable, ok := subRetrieved.(WatchableRetrieved); ok {
			// This is a Watchable sub-retrieved.
			// Start watching for updates from this WatchableRetrieved.
			watchableCount++
			go func() {
				err := watchable.WatchForUpdate()
				select {
				case readyWatchableCh <- readyWatchable{watchable, err}:
					// We have an update, signal it to the outer func.
				case <-cr.watchContext.Done():
					// We are instructed to terminate. Just return.
				}
			}()
		}
	}

	select {
	case readyWatchable := <-readyWatchableCh:

		// At least one watcher fired. Close all sub-Retrieved. We don't need
		// to close the one that fired the signal (readyWatchable.watchable),
		// so provide it as an exception to closeAllExcept().
		if err := cr.closeAllExcept(cr.watchContext, readyWatchable.watchable); err != nil {
			return err
		}

		// Wait for all sub-retrieved WatchForUpdate() to return. They either will return
		// because there is an update or because we closed them via closeAllExcept()
		// call above.
		err := readyWatchable.returnedError
		for i := 0; i < watchableCount-1; i++ {
			select {
			case watchable := <-readyWatchableCh:
				// One more sub-retrieved's WatchForUpdate() returned.
				// ErrSessionClosed is expected because we called Close. If it is
				// any other error then it is unexpected so return the error to the caller.
				if watchable.returnedError != nil && watchable.returnedError != configsource.ErrSessionClosed {
					err = multierr.Append(err, watchable.returnedError)
				}
				continue
			case <-cr.watchContext.Done():
				// We are instructed to terminate.
				return configsource.ErrSessionClosed
			}
		}

		if err == nil {
			// All Watchables returned without error, which means they are ready for
			// another Get() call. Call Get() for all subproviders again and combine
			// the results.
			configMap, err2 := cr.subRetrieved.getAllAndCompose(cr.composeFunc)
			if err2 != nil {
				return err2
			}

			// Remember the result.
			cr.composedMap = configMap
		}

		// Propagate the result to our caller.
		return err

	case <-cr.watchContext.Done():
		// We are instructed to terminate.
		return configsource.ErrSessionClosed
	}
}

// Close all watchable subproviders' Retrieved, except the one that is provided as
// and "except" parameter.
func (cr *compositeRetrieved) closeAllExcept(ctx context.Context, except WatchableRetrieved) error {
	var errs error
	for _, retr := range cr.subRetrieved {
		if watchable, ok := retr.(WatchableRetrieved); ok {
			if watchable != except {
				if err := watchable.Close(ctx); err != nil {
					errs = multierr.Append(errs, err)
				}
			}
		}
	}
	return errs
}

func (cr *compositeRetrieved) Close(ctx context.Context) error {
	// Cancel any long-running operations in WatchForUpdate().
	cr.cancelWatchContext()

	// Close all sub-retrieved.
	return cr.closeAllExcept(ctx, nil)
}
