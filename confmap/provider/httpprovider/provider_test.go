// Copyright The OpenTelemetry Authors
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

package httpprovider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

// Create a provider mocking httpmapprovider works in normal cases
func NewTestProvider() confmap.Provider {
	return &provider{}
}

// Create a provider mocking httpmapprovider works when the returned config file is invalid
func NewTestInvalidProvider() confmap.Provider {
	return &provider{}
}

// Create a provider mocking httpmapprovider works when there is no corresponding config file according to the given http-uri
func NewTestNonExistProvider() confmap.Provider {
	return &provider{}
}

func TestFunctionalityDownloadFileHTTP(t *testing.T) {
	fp := NewTestProvider()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.ReadFile("./testdata/otel-config.yaml")
		if err != nil {
			w.WriteHeader(404)
			_, innerErr := w.Write([]byte("Cannot find the config file"))
			if innerErr != nil {
				fmt.Println("Write failed: ", innerErr)
			}
			return
		}
		w.WriteHeader(200)
		_, err = w.Write(f)
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.NoError(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestUnsupportedScheme(t *testing.T) {
	fp := NewTestProvider()
	_, err := fp.Retrieve(context.Background(), "https://...", nil)
	assert.Error(t, err)
	assert.NoError(t, fp.Shutdown(context.Background()))
}

func TestEmptyURI(t *testing.T) {
	fp := NewTestProvider()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		_, err := w.Write([]byte(""))
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	require.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestNonExistent(t *testing.T) {
	fp := NewTestNonExistProvider()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.ReadFile("./testdata/nonexist-otel-config.yaml")
		if err != nil {
			w.WriteHeader(404)
			_, innerErr := w.Write([]byte("Cannot find the config file"))
			if innerErr != nil {
				fmt.Println("Write failed: ", innerErr)
			}
			return
		}
		w.WriteHeader(200)
		_, err = w.Write(f)
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestInvalidYAML(t *testing.T) {
	fp := NewTestInvalidProvider()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("wrong : ["))
		if err != nil {
			fmt.Println("Write failed: ", err)
		}
	}))
	defer ts.Close()
	_, err := fp.Retrieve(context.Background(), ts.URL, nil)
	assert.Error(t, err)
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestScheme(t *testing.T) {
	fp := NewTestProvider()
	assert.Equal(t, "http", fp.Scheme())
	require.NoError(t, fp.Shutdown(context.Background()))
}

func TestValidateProviderScheme(t *testing.T) {
	assert.NoError(t, confmaptest.ValidateProviderScheme(New()))
}
