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

package filestorage

import (
	"context"
	"errors"
	"time"

	"go.etcd.io/bbolt"
)

var defaultBucket = []byte(`default`)

type fileStorageClient struct {
	db *bbolt.DB
}

func newClient(filePath string, timeout time.Duration) (*fileStorageClient, error) {
	options := &bbolt.Options{
		Timeout: timeout,
		NoSync:  true,
	}
	db, err := bbolt.Open(filePath, 0600, options)
	if err != nil {
		return nil, err
	}

	initBucket := func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
	}
	if err := db.Update(initBucket); err != nil {
		return nil, err
	}

	return &fileStorageClient{db}, nil
}

// Get will retrieve data from storage that corresponds to the specified key
func (c *fileStorageClient) Get(_ context.Context, key string) ([]byte, error) {
	var result []byte
	get := func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errors.New("storage not initialized")
		}
		result = bucket.Get([]byte(key))
		return nil // no error
	}

	if err := c.db.Update(get); err != nil {
		return nil, err
	}
	return result, nil
}

// Set will store data. The data can be retrieved using the same key
func (c *fileStorageClient) Set(_ context.Context, key string, value []byte) error {
	set := func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errors.New("storage not initialized")
		}
		return bucket.Put([]byte(key), value)
	}

	return c.db.Update(set)
}

// Delete will delete data associated with the specified key
func (c *fileStorageClient) Delete(_ context.Context, key string) error {
	delete := func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(defaultBucket)
		if bucket == nil {
			return errors.New("storage not initialized")
		}
		return bucket.Delete([]byte(key))
	}

	return c.db.Update(delete)
}

// Close will close the database
func (c *fileStorageClient) close() error {
	return c.db.Close()
}
