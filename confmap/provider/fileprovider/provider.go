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

package fileprovider // import "go.opentelemetry.io/collector/confmap/provider/fileprovider"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/internal"
)

const (
	schemeName       = "file"
	filePollInterval = time.Second
)

type provider struct {
	eventHandlers    sync.WaitGroup
	eventCloseCh     chan struct{}
	filePollInterval time.Duration
}

// New returns a new confmap.Provider that reads the configuration from a file.
//
// This Provider supports "file" scheme, and can be called with a "uri" that follows:
//
//	file-uri		= "file:" local-path
//	local-path		= [ drive-letter ] file-path
//	drive-letter	= ALPHA ":"
//
// The "file-path" can be relative or absolute, and it can be any OS supported format.
//
// Examples:
// `file:path/to/file` - relative path (unix, windows)
// `file:/path/to/file` - absolute path (unix, windows)
// `file:c:/path/to/file` - absolute path including drive-letter (windows)
// `file:c:\path\to\file` - absolute path including drive-letter (windows)
func New() confmap.Provider {
	return &provider{
		filePollInterval: filePollInterval,
		eventCloseCh:     make(chan struct{}),
	}
}

func (fmp *provider) Retrieve(_ context.Context, uri string, watcherFunc confmap.WatcherFunc) (*confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return nil, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	// Clean the path before using it.
	cleanedPath := filepath.Clean(uri[len(schemeName)+1:])

	// set up the file watcher if the caller asked for watch updates
	// do this before reading the file to avoid a race condition where the file is changed after
	// reading the contents but before the watch is started
	opts := []confmap.RetrievedOption{}
	if watcherFunc != nil {
		closeFunc, err := fmp.startFileWatcher(cleanedPath, watcherFunc)
		if err != nil {
			return &confmap.Retrieved{}, fmt.Errorf("unable to start file watcher %v: %w", uri, err)
		}

		opts = append(opts, confmap.WithRetrievedClose(closeFunc))
	}

	content, err := os.ReadFile(cleanedPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read the file %v: %w", uri, err)
	}

	return internal.NewRetrievedFromYAML(content, opts...)
}

// startFileWatcher creates and starts the watcher for the provided file
func (fmp *provider) startFileWatcher(path string, watcherFunc confmap.WatcherFunc) (confmap.CloseFunc, error) {
	fileWatcher := newPollingFileWatcher(path, fmp.filePollInterval)
	err := fileWatcher.Start()
	if err != nil {
		return nil, err
	}

	closeFunc := func(_ context.Context) error {
		// this is idempotent and safe to call multiple times
		fileWatcher.Close()
		return nil
	}
	// handle events coming in from the watcher
	fmp.eventHandlers.Add(1)
	go func() {
		defer fmp.eventHandlers.Done()
		fmp.handleEventsFromWatcher(fileWatcher, watcherFunc)
	}()

	return closeFunc, nil
}

// handleEventsFromWatcher reads change events from the file watcher's event channel and calls
// the watchFunc on each event
func (fmp *provider) handleEventsFromWatcher(fileWatcher *pollingFileWatcher, watchFunc confmap.WatcherFunc) {
	for { // this loop ends when either the fileWatcher or closeCh is closed
		select {
		case changeEvent, ok := <-fileWatcher.Events:
			if !ok {
				return
			}
			go watchFunc(changeEvent)
		case _, ok := <-fmp.eventCloseCh:
			if !ok {
				// caller called Shutdown before all the closeFuncs, clean up anyway
				fileWatcher.Close()
				return
			}
		}
	}
}

func (*provider) Scheme() string {
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	// according to the spec, the caller should call the closeFunc for all values returned by Retrieve before
	// calling Shutdown, so no handlers should be running at this point
	// just in case, we close this channel to stop any remaining ones
	close(fmp.eventCloseCh)
	fmp.eventHandlers.Wait()
	return nil
}
