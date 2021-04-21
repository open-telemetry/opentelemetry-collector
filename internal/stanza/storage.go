// Copyright The OpenTelemetry Authors
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

package stanza

import (
	"context"
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"

	"go.opentelemetry.io/collector/component"
)

func (r *receiver) setStorageClient(ctx context.Context, host component.Host) error {
	var storageExtension storage.Extension
	for _, ext := range host.GetExtensions() {
		if se, ok := ext.(storage.Extension); ok {
			if storageExtension != nil {
				return errors.New("multiple storage extensions found")
			}
			storageExtension = se
		}
	}

	if storageExtension == nil {
		r.storageClient = storage.NewNopClient()
		return nil
	}

	client, err := storageExtension.GetClient(ctx, component.KindReceiver, r.NamedEntity)
	if err != nil {
		return err
	}

	r.storageClient = client
	return nil
}

func (r *receiver) getPersister() operator.Persister {
	return &persister{r.storageClient}
}

type persister struct {
	client storage.Client
}

var _ operator.Persister = &persister{}

func (p *persister) Get(ctx context.Context, key string) ([]byte, error) {
	return p.client.Get(ctx, key)
}

func (p *persister) Set(ctx context.Context, key string, value []byte) error {
	return p.client.Set(ctx, key, value)
}

func (p *persister) Delete(ctx context.Context, key string) error {
	return p.client.Delete(ctx, key)
}
