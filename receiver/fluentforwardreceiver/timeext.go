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

package fluentforwardreceiver

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/tinylib/msgp/msgp"
)

type eventTimeExt time.Time

func init() {
	msgp.RegisterExtension(0, func() msgp.Extension { return new(eventTimeExt) })
}

func (*eventTimeExt) ExtensionType() int8 {
	return 0x00
}

func (e *eventTimeExt) Len() int {
	return 8
}

func (e *eventTimeExt) MarshalBinaryTo(b []byte) error {
	binary.BigEndian.PutUint32(b[0:], uint32(time.Time(*e).Unix()))
	binary.BigEndian.PutUint32(b[4:], uint32(time.Time(*e).Nanosecond()))

	return nil
}

func (e *eventTimeExt) UnmarshalBinary(b []byte) error {
	if len(b) != 8 {
		return errors.New("data should be exactly 8 bytes")
	}
	secs := int64(binary.BigEndian.Uint32(b[0:]))
	nanos := int64(binary.BigEndian.Uint32(b[4:]))
	*e = eventTimeExt(time.Unix(secs, nanos))
	return nil
}
