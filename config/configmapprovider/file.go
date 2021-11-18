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
	"errors"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/collector/config"
)

var errNoOp = errors.New("no change")

type fileMapProvider struct {
	fileName string
	watching bool
}

// NewFile returns a new Provider that reads the configuration from the given file.
func NewFile(fileName string) Provider {
	return &fileMapProvider{
		fileName: fileName,
	}
}

func (fmp *fileMapProvider) Retrieve(ctx context.Context, onChange func(*ChangeEvent)) (Retrieved, error) {
	if fmp.fileName == "" {
		return nil, errors.New("config file not specified")
	}

	cp, err := config.NewMapFromFile(fmp.fileName)
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %w", fmp.fileName, err)
	}

	// Ensure that watches are only created once as Retrieve is called each time that `onChange()` is invoked.
	if !fmp.watching {
		watchFile(ctx, fmp.fileName, onChange)
		fmp.watching = true
	}

	return &simpleRetrieved{confMap: cp}, nil
}

func (*fileMapProvider) Shutdown(_ context.Context) error {
	return nil
}

// watchFile sets up a watch on a filename to detect changes and calls onChange() with a suitable ChangeEvent.
// The watch is cancelled when the context is done.
func watchFile(ctx context.Context, filename string, onChange func(*ChangeEvent)) {
	var (
		lastfi os.FileInfo
	)

	check := func() error {
		currfi, err := os.Stat(filename)
		modified := false
		if err != nil {
			if os.IsNotExist(err) && lastfi != nil {
				return errNoOp
			}
			return err
		}
		if lastfi != nil && (currfi.Size() != lastfi.Size() || !currfi.ModTime().Equal(lastfi.ModTime())) {
			modified = true
		}

		lastfi = currfi
		if modified {
			return nil
		}

		return errNoOp
	}

	// Check the file every second and if it's been updated then initiate a config reload.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// If check returns a valid event, exit the loop. A new watch will be placed on the next Retrieve()
				err := check()
				if err == nil || err != errNoOp {
					time.Sleep(time.Second * 2)
					onChange(&ChangeEvent{
						Error: err,
					})
				}
			}
		}
	}()
}
