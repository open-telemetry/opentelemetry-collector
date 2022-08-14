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
	"bytes"
	"crypto/sha256"
	"os"
	"sync"
	"time"

	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/confmap"
)

// pollingFileWatcher watches for changes to a file and delivers change events for each one
// It keeps track of the state of the watched file in a simple polling loop
//
// The implementation intentionally doesn't use notification mechanisms provided by the OS
// such as inotify. The reasons for this choice are:
//
//   - Reliability
//     Inotify and similar mechanisms don't work in some circumstances, and require hacky workarounds
//     in others. Problematic cases include Kubernetes ConfigMaps and network filesystems in general.
//   - Simplicity
//     A poll is easy to understand and only uses very basic OS filesystem features.
//   - The advantages of inotify, like performance and the ability to watch whole folders for changes,
//     are irrelevant here. We're only watching a single small file, and we don't need high update
//     frequency.

type pollingFileWatcher struct {
	Events          chan *confmap.ChangeEvent
	pollInterval    time.Duration
	filePath        string
	fileFingerprint []byte
	fileInfo        os.FileInfo
	running         atomic.Bool // this is here to make Close() idempotent and safe to call multiple times
	closeCh         chan struct{}
	wg              sync.WaitGroup
}

func newPollingFileWatcher(filePath string, pollInterval time.Duration) *pollingFileWatcher {
	return &pollingFileWatcher{
		filePath:     filePath,
		pollInterval: pollInterval,
	}
}

// Start starts the watcher. It's safe to call multiple times - it's a noop if the watcher is already
// running. It is, however, not thread-safe.
func (fw *pollingFileWatcher) Start() error {
	var err error

	if fw.running.Load() {
		return nil
	}

	fw.fileInfo, err = os.Stat(fw.filePath)
	if err != nil {
		return err
	}

	fileContent, err := os.ReadFile(fw.filePath)
	if err != nil {
		return err
	}

	fw.fileFingerprint = calculateFingerprint(fileContent)

	fw.closeCh = make(chan struct{})
	fw.Events = make(chan *confmap.ChangeEvent)

	fw.wg.Add(1)
	go func() {
		defer fw.wg.Done()
		fw.startWatchLoop()
	}()

	fw.running.Store(true)

	return nil
}

// startWatchLoop starts the watch loop, which checks for changes to the file every poll interval
// and emits events if needed
// returns after closeCh is closed or after the context is cancelled
func (fw *pollingFileWatcher) startWatchLoop() {
	ticker := time.NewTicker(fw.pollInterval)
	for {
		select {
		case <-ticker.C:
			changeEvent := fw.checkFileForChanges()
			if changeEvent != nil {
				fw.sendEvent(changeEvent)
			}
		case _, ok := <-fw.closeCh:
			if !ok {
				return
			}
		}

	}
}

func (fw *pollingFileWatcher) sendEvent(event *confmap.ChangeEvent) {
	select {
	case fw.Events <- event:
	case _, ok := <-fw.closeCh:
		if !ok {
			return
		}
	}
}

// checkFileForChanges checks if the watched file changed since the previous check and returns an appropriate
// ChangeEvent if it did, and nil if no changes were detected
func (fw *pollingFileWatcher) checkFileForChanges() *confmap.ChangeEvent {
	// check fileinfo first, it's cheaper than reading the content
	fileInfo, err := os.Stat(fw.filePath)
	if err != nil {
		return &confmap.ChangeEvent{Error: err}
	}
	if metadataEqual(fileInfo, fw.fileInfo) {
		return nil
	}
	fw.fileInfo = fileInfo

	fileContent, err := os.ReadFile(fw.filePath)
	if err != nil {
		return &confmap.ChangeEvent{Error: err}
	}

	// check fingerprints
	currentFingerprint := calculateFingerprint(fileContent)
	if bytes.Equal(fw.fileFingerprint, currentFingerprint) {
		return nil
	}
	fw.fileFingerprint = currentFingerprint

	return &confmap.ChangeEvent{}
}

// Close stops the watcher. It's thread-safe and can be safely called multiple times.
func (fw *pollingFileWatcher) Close() {
	if !fw.running.CompareAndSwap(true, false) { // this is here to make it safe to call Close multiple times
		return
	}
	close(fw.closeCh)
	fw.wg.Wait()
	close(fw.Events)
}

// calculateFingerprint calculates the fingerprint used to check for file changes
func calculateFingerprint(content []byte) []byte {
	hash := sha256.New()
	hash.Write(content)
	return hash.Sum(nil)
}

func metadataEqual(first os.FileInfo, second os.FileInfo) bool {
	return first.Size() == second.Size() && first.ModTime() == second.ModTime()
}
