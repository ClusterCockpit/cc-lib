// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sinks

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type httpSinkResponseBody struct {
	io.Reader
	closed bool
}

func (body *httpSinkResponseBody) Close() error {
	body.closed = true
	return nil
}

type httpSinkTestTransport struct {
	body   *httpSinkResponseBody
	status int
}

func (transport httpSinkTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: transport.status, Status: http.StatusText(transport.status),
		Body: transport.body, Header: make(http.Header), Request: request,
	}, nil
}

func TestHttpSinkFlushClosesResponseBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			sink, err := NewHttpSink("audit-body-close", []byte(`{"url":"http://audit.invalid","flush_delay":"0s"}`))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(sink.Close)
			body := &httpSinkResponseBody{Reader: strings.NewReader("synthetic response body")}
			sink.(*HttpSink).client.Transport = httpSinkTestTransport{body: body, status: status}
			messages, err := gen_messages(1)
			if err != nil {
				t.Fatal(err)
			}
			err = sink.Write(messages[0])
			if (err != nil) != (status != http.StatusOK) {
				t.Fatalf("HTTP status error changed for %d: %v", status, err)
			}
			if !body.closed {
				t.Fatal("HTTP response body was not closed after Flush")
			}
		})
	}
}
