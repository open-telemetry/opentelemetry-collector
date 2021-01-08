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

import "github.com/tinylib/msgp/msgp"

type AckResponse struct {
	Ack string `msg:"ack"`
}

func (z AckResponse) EncodeMsg(en *msgp.Writer) error {
	// map header, size 1
	// write "ack"
	err := en.Append(0x81, 0xa3, 0x61, 0x63, 0x6b)
	if err != nil {
		return err
	}

	err = en.WriteString(z.Ack)
	if err != nil {
		return msgp.WrapError(err, "Ack")
	}
	return nil
}
