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

package fileconfigsource

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/fsnotify/fsnotify"

	"go.opentelemetry.io/collector/experimental/configsource"
)

// fileConfigSource retrieves the contents of a file as it is and injects
// it into the configuration as a string.
type fileConfigSource struct{}

var _ (configsource.ConfigSource) = (*fileConfigSource)(nil)

func (e *fileConfigSource) NewSession(context.Context) (configsource.Session, error) {
	return &fileSession{}, nil
}

type fileSession struct {
	watcher *fsnotify.Watcher
}

func (fs *fileSession) Retrieve(_ context.Context, selector string, _ interface{}) (configsource.Retrieved, error) {
	// TODO: just for tests actual implementation needs to lock the file until the contents are read and it is added to the watcher.
	bytes, err := ioutil.ReadFile(selector) // nolint: gosec
	if err != nil {
		return nil, err
	}

	text := string(bytes)
	text = strings.Trim(text, "\n")

	watchForUpdateFn := configsource.WatcherNotSupported
	if fs.watcher == nil {
		fs.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return nil, err
		}

		// First time create a real watch for update function.
		watchForUpdateFn = func() error {
			for {
				select {
				case event, ok := <-fs.watcher.Events:
					if !ok {
						return configsource.ErrSessionClosed
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						return fmt.Errorf("file used in the config modified: %q: %w", event.Name, configsource.ErrValueUpdated)
					}
				case watcherErr, ok := <-fs.watcher.Errors:
					if !ok {
						return configsource.ErrSessionClosed
					}
					return watcherErr
				}
			}
		}
	}

	if err = fs.watcher.Add(selector); err != nil {
		return nil, err
	}

	return configsource.NewRetrieved(text, watchForUpdateFn), nil
}

func (fs *fileSession) RetrieveEnd(context.Context) error {
	return nil
}

func (fs *fileSession) Close(context.Context) error {
	if fs.watcher != nil {
		return fs.watcher.Close()
	}

	return nil
}
