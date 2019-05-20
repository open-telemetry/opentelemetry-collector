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
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

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
	numErrors := len(errs)
	if numErrors == 1 {
		return errs[0]
	} else if numErrors > 1 {
		errMsgs := make([]string, 0, numErrors)
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("[%s]", strings.Join(errMsgs, "; "))
	}
	return nil
}
