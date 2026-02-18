// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package lrucache

import (
	"bytes"
	"maps"
	"net/http"
	"strconv"
	"time"
)

// HttpHandler is an HTTP middleware that caches HTTP responses using an LRU cache.
//
// It can be used to cache static assets, API responses, or any GET requests.
// By default, the request's RequestURI is used as the cache key.
//
// Caching behavior:
//   - Only GET requests are cached (other methods pass through)
//   - Responses with status code 200 are cached with the configured TTL
//   - Responses with other status codes are cached with TTL=0 (immediate re-fetch)
//   - If the response sets an "Expires" header, it overrides the default TTL
//   - The "Age" header is automatically set to indicate cache age
//
// The CacheKey function can be customized to change how cache keys are generated
// from requests (e.g., to include query parameters or headers).
type HttpHandler struct {
	cache      *Cache        // LRU cache instance
	fetcher    http.Handler  // Next handler in the chain
	defaultTTL time.Duration // Default time-to-live for cached responses

	// CacheKey allows overriding the way the cache key is extracted
	// from the http request. The default is to use the RequestURI.
	CacheKey func(*http.Request) string
}

var _ http.Handler = (*HttpHandler)(nil)

// cachedResponseWriter wraps an http.ResponseWriter to capture the response
// for caching. It buffers the response body and status code.
type cachedResponseWriter struct {
	w          http.ResponseWriter // Original response writer
	statusCode int                 // HTTP status code
	buf        bytes.Buffer        // Buffered response body
}

// cachedResponse represents a cached HTTP response.
type cachedResponse struct {
	headers    http.Header // Response headers
	statusCode int         // HTTP status code
	data       []byte      // Response body
	fetched    time.Time   // When this response was fetched
}

var _ http.ResponseWriter = (*cachedResponseWriter)(nil)

func (crw *cachedResponseWriter) Header() http.Header {
	return crw.w.Header()
}

func (crw *cachedResponseWriter) Write(bytes []byte) (int, error) {
	return crw.buf.Write(bytes)
}

func (crw *cachedResponseWriter) WriteHeader(statusCode int) {
	crw.statusCode = statusCode
}

// NewHttpHandler creates a new caching HTTP handler.
//
// The handler caches responses from the fetcher handler. If no cached response
// is found or it has expired, fetcher is called to generate the response.
//
// If the fetcher sets an "Expires" header, the TTL is calculated from that header.
// Otherwise, the default TTL is used. Responses with status codes other than 200
// are cached with TTL=0 (immediate expiration).
//
// Parameters:
//   - maxmemory: Maximum cache size in bytes (size of response bodies)
//   - ttl: Default time-to-live for cached responses
//   - fetcher: The handler to call when cache misses occur
//
// Example:
//
//	// Cache static files for 1 hour, max 100MB
//	fileServer := http.FileServer(http.Dir("./static"))
//	cachedHandler := lrucache.NewHttpHandler(100*1024*1024, 1*time.Hour, fileServer)
//	http.Handle("/static/", cachedHandler)
func NewHttpHandler(maxmemory int, ttl time.Duration, fetcher http.Handler) *HttpHandler {
	return &HttpHandler{
		cache:      New(maxmemory),
		defaultTTL: ttl,
		fetcher:    fetcher,
		CacheKey: func(r *http.Request) string {
			return r.RequestURI
		},
	}
}

// NewMiddleware returns a gorilla/mux style middleware function.
//
// This is a convenience wrapper around NewHttpHandler for use with middleware chains.
//
// Example:
//
//	r := mux.NewRouter()
//	r.Use(lrucache.NewMiddleware(100*1024*1024, 1*time.Hour))
func NewMiddleware(maxmemory int, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return NewHttpHandler(maxmemory, ttl, next)
	}
}

// ServeHTTP implements the http.Handler interface.
//
// It attempts to serve the response from cache. If not cached or expired,
// it calls the fetcher handler and caches the result.
//
// Only GET requests are cached; other HTTP methods pass through to the fetcher.
func (h *HttpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.ServeHTTP(rw, r)
		return
	}

	cr := h.cache.Get(h.CacheKey(r), func() (any, time.Duration, int) {
		crw := &cachedResponseWriter{
			w:          rw,
			statusCode: 200,
			buf:        bytes.Buffer{},
		}

		h.fetcher.ServeHTTP(crw, r)

		cr := &cachedResponse{
			headers:    rw.Header().Clone(),
			statusCode: crw.statusCode,
			data:       crw.buf.Bytes(),
			fetched:    time.Now(),
		}
		cr.headers.Set("Content-Length", strconv.Itoa(len(cr.data)))

		ttl := h.defaultTTL
		if cr.statusCode != http.StatusOK {
			ttl = 0
		} else if cr.headers.Get("Expires") != "" {
			if expires, err := http.ParseTime(cr.headers.Get("Expires")); err == nil {
				ttl = time.Until(expires)
			}
		}

		return cr, ttl, len(cr.data)
	}).(*cachedResponse)

	maps.Copy(rw.Header(), cr.headers)

	cr.headers.Set("Age", strconv.Itoa(int(time.Since(cr.fetched).Seconds())))

	rw.WriteHeader(cr.statusCode)
	rw.Write(cr.data)
}
