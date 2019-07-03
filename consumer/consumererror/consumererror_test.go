// Copyright 2019, OpenTelemetry Authors
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

package consumererror

import (
	"errors"
	"testing"
)

func TestPermanent(t *testing.T) {
	err := errors.New("testError")
	if IsPermanent(err) {
		t.Fatalf("IsPermanent() = true, want false")
	}
	err = Permanent(err)
	if !IsPermanent(err) {
		t.Fatalf("IsPermanent() = false, want true")
	}
}

func TestIsPermanent_NilError(t *testing.T) {
	var err error
	if IsPermanent(err) {
		t.Fatalf("IsPermanent() = true, want false")
	}
}
