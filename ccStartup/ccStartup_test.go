package startup

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	topo "github.com/ClusterCockpit/cc-lib/v2/ccTopology"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func TestEmptyConfig(t *testing.T) {
	config := []byte("{}")

	err := CCStartup(config)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestNoTopoConfig(t *testing.T) {
	config := []byte(`{"send_topology" : false}`)

	err := CCStartup(config)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestHttp(t *testing.T) {
	var wg sync.WaitGroup
	httpReceiver := func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			buf, err := io.ReadAll(r.Body)
			if err == nil {
				topol, err := topo.RemoteTopology(buf)
				if err != nil {
					t.Errorf("received topology not valid: %s", err.Error())
					t.Error(string(buf))
				} else {
					t.Logf("received valid topology of sytem with %d hwthreads", len(topol.GetHwthreads()))
					t.Log(string(buf))
				}
			}
		}
	}
	startupConfig := `{"http" : { "url": "http://localhost:8082"}}`
	t.Logf("startup configuration: %s", startupConfig)
	// Create http server
	httpserver := &http.Server{
		Addr:        "localhost:8082",
		Handler:     nil, // handler to invoke, http.DefaultServeMux if nil
		IdleTimeout: time.Duration(10 * time.Second),
	}
	http.HandleFunc("/", httpReceiver)

	wg.Go(func() {
		t.Logf("starting HTTP server on: %s", "localhost:8082")
		err := httpserver.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			t.Errorf("failed to listen and serve at %s", "localhost:8082")
			wg.Done()
			return
		}
	})
	t.Log("running CCStartup")
	err := CCStartup(json.RawMessage(startupConfig))
	if err != nil {
		t.Error(err.Error())
	}
	t.Log("shutdown HTTP server")
	httpserver.Shutdown(context.Background())
	wg.Wait()
}

func TestNats(t *testing.T) {
	startupConfig := `{"nats" : { "url": "http://localhost:4222", "subject" : "topology"}}`
	t.Logf("startup configuration: %s", startupConfig)

	opts := &server.Options{
		Host: "localhost",
		Port: 4222,
	}
	uri := "http://localhost:4222"
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Errorf("nats server cannot be created for %s", uri)
		return
	}
	t.Logf("starting nats server at %s", uri)
	server.Run(ns)
	if !ns.ReadyForConnections(4 * time.Second) {
		t.Errorf("nats server not ready for connection after %d seconds", 4)
		return
	}
	t.Logf("connecting nats client to %s", uri)
	c, err := nats.Connect(uri, nil)
	if err != nil {
		t.Errorf("failed to connect to nats server %s: %s", uri, err.Error())
		return
	}
	sub, err := c.Subscribe("topology", func(msg *nats.Msg) {
		topol, err := topo.RemoteTopology(msg.Data)
		if err != nil {
			t.Errorf("received topology not valid: %s", err.Error())
			t.Error(string(msg.Data))
		} else {
			t.Logf("received valid topology of sytem with %d hwthreads", len(topol.GetHwthreads()))
			t.Log(string(msg.Data))
		}
	})

	t.Log("running CCStartup")
	err = CCStartup(json.RawMessage(startupConfig))
	if err != nil {
		t.Error(err.Error())
	}
	time.Sleep(time.Second)

	t.Log("shutdown nats client")
	sub.Unsubscribe()
	c.Close()

	t.Log("shutdown nats server")
	ns.Shutdown()
}
