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

package receiver

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestToGrpcServerOption(t *testing.T) {
	type testCase struct {
		in  TLSCredentials
		err error
	}

	testCases := []testCase{
		{
			in: TLSCredentials{
				CertFile: "/badpath",
				KeyFile:  "/badpath",
			},
			err: &os.PathError{
				Op:   "open",
				Path: "/badpath",
				Err:  unix.ENOENT,
			},
		},
		{
			in: TLSCredentials{
				CertFile: path.Join(".", "jaegerreceiver/testdata", "certificate.pem"),
				KeyFile:  "/badpath",
			},
			err: &os.PathError{
				Op:   "open",
				Path: "/badpath",
				Err:  unix.ENOENT,
			},
		},
		{
			in: TLSCredentials{
				CertFile: path.Join(".", "jaegerreceiver/testdata", "certificate.pem"),
				KeyFile:  path.Join(".", "jaegerreceiver/testdata", "key.pem"),
			},
			err: nil,
		},
	}

	for _, c := range testCases {
		_, err := c.in.ToGrpcServerOption()
		assert.Equal(t, c.err, err)
	}
}
