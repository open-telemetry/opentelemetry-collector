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

package exporterhelper

import (
	"context"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// request is an abstraction of an individual request (batch of data) independent of the type of the data (traces, metrics, logs).
type request interface {
	// context returns the Context of the requests.
	context() context.Context
	// setContext updates the Context of the requests.
	setContext(context.Context)
	export(ctx context.Context) error
	// Returns a new request may contain the items left to be sent if some items failed to process and can be retried.
	// Otherwise, it should return the original request.
	onError(error) request
	// Returns the count of spans/metric points or log records.
	count() int

	// PersistentRequest provides interface with additional capabilities required by persistent queue
	internal.PersistentRequest
}

// requestSender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).
type requestSender interface {
	send(req request) error
}

// baseRequest is a base implementation for the request.
type baseRequest struct {
	ctx                        context.Context
	processingFinishedCallback func()
}

func (req *baseRequest) context() context.Context {
	return req.ctx
}

func (req *baseRequest) setContext(ctx context.Context) {
	req.ctx = ctx
}

func (req *baseRequest) SetOnProcessingFinished(callback func()) {
	req.processingFinishedCallback = callback
}

func (req *baseRequest) OnProcessingFinished() {
	if req.processingFinishedCallback != nil {
		req.processingFinishedCallback()
	}
}
