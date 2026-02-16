package sinks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

var testHttpConfig = HttpSinkConfig{
	URL:             "http://localhost:8082/",
	Timeout:         "1s",
	IdleConnTimeout: "10s",
	Precision:       "ns",
}

var receivedHttpMessages = make([]string, 0)

func httpReceiver(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		buf, err := io.ReadAll(r.Body)
		if err == nil {
			lines := strings.Split(string(buf), "\n")
			for _, l := range lines {
				if len(l) > 0 {
					receivedHttpMessages = append(receivedHttpMessages, l)
				}
			}
		}
	}
}

func TestHttpSink(t *testing.T) {
	var wg sync.WaitGroup
	jsonConfig, err := json.Marshal(testHttpConfig)
	if err != nil {
		t.Errorf("failed to marshal configuration: %s", err.Error())
		return
	}
	receivedHttpMessages = receivedHttpMessages[:0]

	t.Logf("setup http sink to %s", testHttpConfig.URL)
	s, err := NewHttpSink("testsink", jsonConfig)
	if err != nil {
		t.Errorf("failed to setup http sink: %s", err.Error())
		return
	}

	// Create http server
	httpserver := &http.Server{
		Addr:        "localhost:8082",
		Handler:     nil, // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout: time.Duration(10 * time.Second),
	}
	http.HandleFunc("/", httpReceiver)

	t.Logf("starting http server listening at %s", httpserver.Addr)
	wg.Add(1)
	go func() {
		err := httpserver.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			t.Errorf("failed to listen and serve at %s", testHttpConfig.URL)
			wg.Done()
			return
		}
		wg.Done()
	}()
	time.Sleep(500 * time.Millisecond)
	msgs, _ := gen_messages(10)
	t.Logf("writing %d messages to sink", len(msgs))
	for _, m := range msgs {
		t.Log(m.String())
		s.Write(m)
	}
	t.Log("flushing sink")
	s.Flush()

	t.Log("shutdown http sink")
	s.Close()

	t.Log("shutdown http server")
	httpserver.Shutdown(context.Background())
	wg.Wait()

	if len(msgs) != len(receivedHttpMessages) {
		t.Error("not all messages sent and received")
		return
	}

	for i, m := range msgs {
		ms := m.ToLineProtocol(nil)
		ms = strings.Trim(ms, "\n")
		if ms != receivedHttpMessages[i] {
			t.Errorf("message %d invalid: '%s' vs '%s'", i, ms, receivedHttpMessages[i])
		}
	}
}
