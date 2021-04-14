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

package configsource

// WatcherNotSupported is the a watcher function that always returns ErrWatcherNotSupported.
func WatcherNotSupported() error {
	return ErrWatcherNotSupported
}

type retrieved struct {
	value            interface{}
	watchForUpdateFn func() error
}

// NewRetrieved is a helper that implements the Retrieved interface.
func NewRetrieved(value interface{}, watchForUpdateFn func() error) Retrieved {
	return &retrieved{
		value,
		watchForUpdateFn,
	}
}

var _ Retrieved = (*retrieved)(nil)

func (r *retrieved) Value() interface{} {
	return r.value
}

func (r *retrieved) WatchForUpdate() error {
	return r.watchForUpdateFn()
}
