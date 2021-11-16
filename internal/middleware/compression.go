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

package middleware // import "go.opentelemetry.io/collector/internal/middleware"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"io"
	"net/http"
)

const (
	headerContentEncoding = "Content-Encoding"
	headerValueGZIP       = "gzip"
	headerValueSnappy	  = "snappy"
	headerValueZstd		  = "zstd"
)

type (
	CompressGzipRoundTripper struct {
		http.RoundTripper
	}

	CompressSnappyRoundTripper struct {
		http.RoundTripper
	}

	CompressZstdRoundTripper struct {
		http.RoundTripper
	}

	noOpReadCloser struct {
		io.Reader
	}
)

func NewCompressGzipRoundTripper(rt http.RoundTripper) *CompressGzipRoundTripper {
	return &CompressGzipRoundTripper{
		RoundTripper: rt,
	}
}

func (r *CompressGzipRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(headerContentEncoding) != "" {
		// If the header already specifies a content encoding then skip compression
		// since we don't want to compress it again. This is a safeguard that normally
		// should not happen since CompressRoundTripper is not intended to be used
		// with http clients which already do their own compression.
		return r.RoundTripper.RoundTrip(req)
	}

	// Gzip the body.
	buf := bytes.NewBuffer([]byte{})
	gzipWriter := gzip.NewWriter(buf)
	_, copyErr := io.Copy(gzipWriter, req.Body)
	closeErr := req.Body.Close()

	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	// Create a new request since the docs say that we cannot modify the "req"
	// (see https://golang.org/pkg/net/http/#RoundTripper).
	cReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), buf)
	if err != nil {
		return nil, err
	}

	// Clone the headers and add gzip encoding header.
	cReq.Header = req.Header.Clone()
	cReq.Header.Add(headerContentEncoding, headerValueGZIP)

	return r.RoundTripper.RoundTrip(cReq)
}

func NewCompressSnappyRoundTripper(rt http.RoundTripper) *CompressSnappyRoundTripper {
	return &CompressSnappyRoundTripper{
		RoundTripper: rt,
	}
}

func (r *CompressSnappyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(headerContentEncoding) != "" {
		// If the header already specifies a content encoding then skip compression
		// since we don't want to compress it again. This is a safeguard that normally
		// should not happen since CompressRoundTripper is not intended to be used
		// with http clients which already do their own compression.
		return r.RoundTripper.RoundTrip(req)
	}

	// Compress the body with Snappy.
	buf := bytes.NewBuffer([]byte{})
	snappyWriter := snappy.NewBufferedWriter(buf)
	_, copyErr := io.Copy(snappyWriter, req.Body)
	closeErr := req.Body.Close()

	if err := snappyWriter.Close(); err != nil {
		return nil, err
	}

	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	// Create a new request since the docs say that we cannot modify the "req"
	// (see https://golang.org/pkg/net/http/#RoundTripper).
	cReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), buf)
	if err != nil {
		return nil, err
	}

	// Clone the headers and add snappy as an encoding header.
	cReq.Header = req.Header.Clone()
	cReq.Header.Add(headerContentEncoding, headerValueSnappy)

	return r.RoundTripper.RoundTrip(cReq)
}

func NewCompressZstdRoundTripper(rt http.RoundTripper) *CompressZstdRoundTripper {
	return &CompressZstdRoundTripper{
		RoundTripper: rt,
	}
}

func (r *CompressZstdRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(headerContentEncoding) != "" {
		// If the header already specifies a content encoding then skip compression
		// since we don't want to compress it again. This is a safeguard that normally
		// should not happen since CompressRoundTripper is not intended to be used
		// with http clients which already do their own compression.
		return r.RoundTripper.RoundTrip(req)
	}

	// Compress the body with zstd.
	buf := bytes.NewBuffer([]byte{})
	zstdWriter, _ := zstd.NewWriter(buf)
	_, copyErr := io.Copy(zstdWriter, req.Body)
	closeErr := req.Body.Close()

	if err := zstdWriter.Close(); err != nil {
		return nil, err
	}

	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	// Create a new request since the docs say that we cannot modify the "req"
	// (see https://golang.org/pkg/net/http/#RoundTripper).
	cReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), buf)
	if err != nil {
		return nil, err
	}

	// Clone the headers and add zstd as an encoding header.
	cReq.Header = req.Header.Clone()
	cReq.Header.Add(headerContentEncoding, headerValueZstd)

	return r.RoundTripper.RoundTrip(cReq)
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)

type decompressor struct {
	errorHandler ErrorHandler
}

type DecompressorOption func(d *decompressor)

func WithErrorHandler(e ErrorHandler) DecompressorOption {
	return func(d *decompressor) {
		d.errorHandler = e
	}
}

// HTTPContentDecompressor is a middleware that offloads the task of handling compressed
// HTTP requests by identifying the compression format in the "Content-Encoding" header and re-writing
// request body so that the handlers further in the chain can work on decompressed data.
// It supports gzip and deflate/zlib compression.
func HTTPContentDecompressor(h http.Handler, opts ...DecompressorOption) http.Handler {
	d := &decompressor{}
	for _, o := range opts {
		o(d)
	}
	if d.errorHandler == nil {
		d.errorHandler = defaultErrorHandler
	}
	return d.wrap(h)
}

func (d *decompressor) wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newBody, err := newBodyReader(r)
		if err != nil {
			d.errorHandler(w, r, err.Error(), http.StatusBadRequest)
			return
		}
		if newBody != nil {
			defer newBody.Close()
			// "Content-Encoding" header is removed to avoid decompressing twice
			// in case the next handler(s) have implemented a similar mechanism.
			r.Header.Del("Content-Encoding")
			// "Content-Length" is set to -1 as the size of the decompressed body is unknown.
			r.Header.Del("Content-Length")
			r.ContentLength = -1
			r.Body = newBody
		}
		h.ServeHTTP(w, r)
	})
}

func newBodyReader(r *http.Request) (io.ReadCloser, error) {
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return gr, nil
	case "deflate", "zlib":
		zr, err := zlib.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return zr, nil
	case "snappy":
		sr := io.NopCloser(snappy.NewReader(r.Body))
		return sr, nil
	case "zstd":
		zr, err := zstd.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		return zr.IOReadCloser(), nil
	}
	return nil, nil
}

// defaultErrorHandler writes the error message in plain text.
func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, errMsg string, statusCode int) {
	http.Error(w, errMsg, statusCode)
}
