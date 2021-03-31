// Copyright  The OpenTelemetry Authors
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

package kafkaexporter

import (
	"github.com/Shopify/sarama"
	"github.com/xdg-go/scram"
)

var _ sarama.SCRAMClient = (*XDGSCRAMClient)(nil)

// XDGSCRAMClient uses xdg-go scram to authentication conversation
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// Begin starts the XDGSCRAMClient conversation.
func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

// Step takes a string provided from a server (or just an empty string for the
// very first conversation step) and attempts to move the authentication
// conversation forward.  It returns a string to be sent to the server or an
// error if the server message is invalid.  Calling Step after a conversation
// completes is also an error.
func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	return x.ClientConversation.Step(challenge)

}

// Done returns true if the conversation is completed or has errored.
func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}
