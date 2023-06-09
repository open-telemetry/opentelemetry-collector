// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This file contains helper functions regarding compression/decompression for confighttp.

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/config/configcompression"
)

type compressRoundTripper struct {
	rt              http.RoundTripper
	compressionType configcompression.CompressionType
	compressor      *compressor
}

func newCompressRoundTripper(rt http.RoundTripper, compressionType configcompression.CompressionType) (*compressRoundTripper, error) {
	encoder, err := newCompressor(compressionType)
	if err != nil {
		return nil, err
	}
	return &compressRoundTripper{
		rt:              rt,
		compressionType: compressionType,
		compressor:      encoder,
	}, nil
}

func (r *compressRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get(headerContentEncoding) != "" {
		// If the header already specifies a content encoding then skip compression
		// since we don't want to compress it again. This is a safeguard that normally
		// should not happen since CompressRoundTripper is not intended to be used
		// with http clients which already do their own compression.
		return r.rt.RoundTrip(req)
	}

	// Compress the body.
	buf := bytes.NewBuffer([]byte{})
	if err := r.compressor.compress(buf, req.Body); err != nil {
		return nil, err
	}

	// Create a new request since the docs say that we cannot modify the "req"
	// (see https://golang.org/pkg/net/http/#RoundTripper).
	cReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), buf)
	if err != nil {
		return nil, err
	}

	// Clone the headers and add the encoding header.
	cReq.Header = req.Header.Clone()
	cReq.Header.Add(headerContentEncoding, string(r.compressionType))

	return r.rt.RoundTrip(cReq)
}

type errorHandler func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)

type decompressor struct {
	errorHandler
}

type decompressorOption func(d *decompressor)

func withErrorHandlerForDecompressor(e errorHandler) decompressorOption {
	return func(d *decompressor) {
		d.errorHandler = e
	}
}

// httpContentDecompressor offloads the task of handling compressed HTTP requests
// by identifying the compression format in the "Content-Encoding" header and re-writing
// request body so that the handlers further in the chain can work on decompressed data.
// It supports gzip and deflate/zlib compression.
func httpContentDecompressor(h http.Handler, opts ...decompressorOption) http.Handler {
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
	}
	return nil, nil
}

// defaultErrorHandler writes the error message in plain text.
func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, errMsg string, statusCode int) {
	http.Error(w, errMsg, statusCode)
}
