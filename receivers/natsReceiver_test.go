package receivers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	server "github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
)

var natsReceiverTestConfig json.RawMessage = json.RawMessage(`{
	"type": "nats",
	"address" : "localhost",
	"port" : "8082",
	"subject" : "test"
}`)

func TestNatsReceiver(t *testing.T) {
	numMessage := 10
	sink := make(chan lp.CCMessage, numMessage)

	opts := &server.Options{
		Host: "localhost",
		Port: 8082,
	}
	uri := fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port)
	t.Logf("starting nats server for %s", uri)
	ns, err := server.NewServer(opts)
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

	r, err := NewNatsReceiver("testreceiver", natsReceiverTestConfig)
	if err != nil {
		t.Errorf("failed to start nats receiver with config '%s': %s", string(httpReceiverTestConfig), err.Error())
		return
	}
	r.SetSink(sink)
	t.Log("Starting nats receiver")
	r.Start()

	msgs := gen_messages(numMessage)
	for _, m := range msgs {
		ilp := m.ToLineProtocol(nil)
		t.Logf("publishing to subject test: %s", ilp)
		err := c.Publish("test", []byte(ilp))
		if err != nil {
			t.Errorf("failed sending '%s': %s", ilp, err.Error())
		}
	}

	time.Sleep(2*time.Second)
	recvm := make([]lp.CCMessage, 0, numMessage)
	for i := 0; i < numMessage && len(sink) > 0; i++ {
		recvm = append(recvm, <-sink)
	}
	if len(recvm) != len(msgs) {
		t.Errorf("received only %d metrics", len(recvm))
	} else {
		for i := 0; i < numMessage; i++ {
			if msgs[i].ToLineProtocol(nil) != recvm[i].ToLineProtocol(nil) {
				t.Errorf("metrics do no match '%s' vs '%s'", msgs[i].ToLineProtocol(nil), recvm[i].ToLineProtocol(nil))
			}
		}
	}

	t.Log("Closing nats receiver")
	r.Close()
	t.Log("Closing nats client")
	c.Close()
	t.Log("Closing nats server")
	ns.Shutdown()
}
