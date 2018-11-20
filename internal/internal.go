// Copyright 2018, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// Service contains metadata about the exporter service.
type Service struct {
	Endpoint string `json:"endpoint"`
}

// WriteToEndpointFile writes service metadata to
// canonical endpoint file.
func (s *Service) WriteToEndpointFile() (path string, err error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	path = defaultEndpointFile()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(path, data, 0755); err != nil {
		return "", err
	}
	return path, nil
}

// ParseEndpointFile reads and parses the canonical endpoint
// file for metadata.
func ParseEndpointFile() (*Service, error) {
	file := defaultEndpointFile()
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// defaultEndpointFile is the location where the
// endpoint file is at on the current platform.
func defaultEndpointFile() string {
	const f = "opencensus.endpoint"
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "opencensus", f)
	}
	return filepath.Join(guessUnixHomeDir(), ".config", f)
}

func guessUnixHomeDir() string {
	// Prefer $HOME over user.Current due to glibc bug: golang.org/issue/13470
	if v := os.Getenv("HOME"); v != "" {
		return v
	}
	// Else, fall back to user.Current:
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}

// TimeToTimestamp converts a time.Time to a timestamp.Timestamp pointer.
func TimeToTimestamp(t time.Time) *timestamp.Timestamp {
	if t.IsZero() {
		return nil
	}
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}

// CombineErrors converts a list of errors into one error.
func CombineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	// Otherwise
	buf := new(strings.Builder)
	for _, err := range errs {
		fmt.Fprintf(buf, "%v\n", err)
	}
	return errors.New(buf.String())
}
