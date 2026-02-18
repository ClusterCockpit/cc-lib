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

var testInfluxAsyncConfig = InfluxAsyncSinkConfig{
	Host:          "localhost",
	Port:          "8082",
	Database:      "testdb",
	Organization:  "testorg",
	FlushInterval: 1,
	Precision:     "ns",
	Password:      "test",
}

var receivedInfluxAsyncMessages = make([]string, 0)

func InfluxAsyncPing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	w.WriteHeader(204)
}

func InfluxAsyncReceiver(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		buf, err := io.ReadAll(r.Body)
		if err == nil {
			lines := strings.SplitSeq(string(buf), "\n")
			for l := range lines {
				if len(l) > 0 {
					receivedInfluxAsyncMessages = append(receivedInfluxAsyncMessages, l)
				}
			}
		}
	}
	w.WriteHeader(204)
}

func TestInfluxAsyncSink(t *testing.T) {
	var wg sync.WaitGroup
	jsonConfig, err := json.Marshal(testInfluxAsyncConfig)
	if err != nil {
		t.Errorf("failed to marshal configuration: %s", err.Error())
		return
	}
	receivedInfluxAsyncMessages = receivedInfluxAsyncMessages[:0]

	// Create http server
	serv := http.NewServeMux()
	httpserver := &http.Server{
		Addr:        "localhost:8082",
		Handler:     serv, // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout: time.Duration(10 * time.Second),
	}
	serv.HandleFunc("/api/v2/write", InfluxAsyncReceiver)
	serv.HandleFunc("/ping", InfluxPing)

	t.Logf("starting http server listening at %s", httpserver.Addr)
	wg.Go(func() {
		err := httpserver.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			t.Errorf("failed to listen and serve at %s:%s", testInfluxAsyncConfig.Host, testInfluxAsyncConfig.Port)
			wg.Done()
			return
		}
	})
	time.Sleep(500 * time.Millisecond)
	t.Logf("setup influx sink to %s:%s organization %s database %s", testInfluxAsyncConfig.Host, testInfluxAsyncConfig.Port, testInfluxAsyncConfig.Organization, testInfluxAsyncConfig.Database)
	s, err := NewInfluxAsyncSink("testsink", jsonConfig)
	if err != nil {
		t.Errorf("failed to setup influx sink: %s", err.Error())
		return
	}

	msgs, _ := gen_messages(10)
	t.Logf("writing %d messages to sink", len(msgs))
	for _, m := range msgs {
		t.Log(m.String())
		s.Write(m)
	}
	t.Log("flushing sink")
	s.Flush()

	t.Log("shutdown influx sink")
	s.Close()

	t.Log("shutdown http server")
	httpserver.Shutdown(context.Background())
	wg.Wait()

	if len(msgs) != len(receivedInfluxAsyncMessages) {
		t.Errorf("not all messages sent and received: %d vs %d", len(msgs), len(receivedInfluxAsyncMessages))
		return
	}

	for i, m := range msgs {
		ms := m.ToLineProtocol(nil)
		ms = strings.Trim(ms, "\n")
		if ms != receivedInfluxAsyncMessages[i] {
			t.Errorf("message %d invalid: '%s' vs '%s'", i, ms, receivedInfluxAsyncMessages[i])
		}
	}
}
