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

type InfluxSinkConfig struct {
	defaultSinkConfig
	Host         string `json:"host,omitempty"`
	Port         string `json:"port,omitempty"`
	Database     string `json:"database,omitempty"`
	User         string `json:"user,omitempty"`
	Password     string `json:"password,omitempty"`
	Organization string `json:"organization,omitempty"`
	SSL          bool   `json:"ssl,omitempty"`
	// Maximum number of points sent to server in single request.
	// Default: 1000
	BatchSize int `json:"batch_size,omitempty"`

	// Time interval for delayed sending of metrics.
	// If the buffers are already filled before the end of this interval,
	// the metrics are sent without further delay.
	// Default: 1s
	FlushInterval string `json:"flush_delay,omitempty"`
	flushDelay    time.Duration

	// Influx client options:

	// HTTP request timeout
	HTTPRequestTimeout string `json:"http_request_timeout"`
	// Retry interval
	InfluxRetryInterval string `json:"retry_interval,omitempty"`
	// maximum delay between each retry attempt
	InfluxMaxRetryInterval string `json:"max_retry_interval,omitempty"`
	// base for the exponential retry delay
	InfluxExponentialBase uint `json:"retry_exponential_base,omitempty"`
	// maximum count of retry attempts of failed writes
	InfluxMaxRetries uint `json:"max_retries,omitempty"`
	// maximum total retry timeout
	InfluxMaxRetryTime string `json:"max_retry_time,omitempty"`
	// Specify whether to use GZip compression in write requests
	InfluxUseGzip bool `json:"use_gzip"`
	// Timestamp precision
	Precision string `json:"precision,omitempty"`
}

var testInfluxConfig = InfluxSinkConfig{
	Host:          "localhost",
	Port:          "8082",
	Database:      "testdb",
	Organization:  "testorg",
	FlushInterval: "1s",
	Precision:     "ns",
	Password:      "test",
}

var receivedInfluxMessages = make([]string, 0)

func InfluxPing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func InfluxReceiver(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {

		buf, err := io.ReadAll(r.Body)
		if err == nil {
			lines := strings.SplitSeq(string(buf), "\n")
			for l := range lines {
				if len(l) > 0 {
					receivedInfluxMessages = append(receivedInfluxMessages, l)
				}
			}
		}
	}
	w.WriteHeader(204)
}

func TestInfluxSink(t *testing.T) {
	var wg sync.WaitGroup
	jsonConfig, err := json.Marshal(testInfluxConfig)
	if err != nil {
		t.Errorf("failed to marshal configuration: %s", err.Error())
		return
	}
	receivedInfluxMessages = receivedInfluxMessages[:0]

	// Create http server
	serv := http.NewServeMux()
	httpserver := &http.Server{
		Addr:        "localhost:8082",
		Handler:     serv, // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout: time.Duration(10 * time.Second),
	}
	serv.HandleFunc("/api/v2/write", InfluxReceiver)
	serv.HandleFunc("/ping", InfluxPing)

	t.Logf("starting http server listening at %s", httpserver.Addr)
	wg.Go(func() {
		err := httpserver.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			t.Errorf("failed to listen and serve at %s:%s", testInfluxConfig.Host, testInfluxConfig.Port)
			wg.Done()
			return
		}
	})
	time.Sleep(500 * time.Millisecond)
	t.Logf("setup influx sink to %s:%s organization %s database %s", testInfluxConfig.Host, testInfluxConfig.Port, testInfluxConfig.Organization, testInfluxConfig.Database)
	s, err := NewInfluxSink("testsink", jsonConfig)
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

	if len(msgs) != len(receivedInfluxMessages) {
		t.Errorf("not all messages sent and received: %d vs %d", len(msgs), len(receivedInfluxMessages))
		return
	}

	for i, m := range msgs {
		ms := m.ToLineProtocol(nil)
		ms = strings.Trim(ms, "\n")
		if ms != receivedInfluxMessages[i] {
			t.Errorf("message %d invalid: '%s' vs '%s'", i, ms, receivedInfluxMessages[i])
		}
	}
}
