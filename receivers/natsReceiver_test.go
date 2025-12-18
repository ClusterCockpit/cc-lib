package receivers

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
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

var serverReady = make(chan bool)
var sendDone = make(chan bool)

func TestNatsReceiver(t *testing.T) {
	numMessage := 10
	sink := make(chan lp.CCMessage, numMessage)
	serverDone := make(chan bool, numMessage)
	var wg sync.WaitGroup

	opts := &server.Options{
		Host: "localhost",
		Port: 8082,
	}
	uri := fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port)

	wg.Add(1)
	go func() {

		t.Logf("starting nats server for %s", uri)
		ns, err := server.NewServer(opts)
		if err != nil {
			t.Errorf("failed to start nats server for %s: %s", uri, err.Error())
			os.Exit(1)
		}
		server.Run(ns)

		if !ns.ReadyForConnections(4 * time.Second) {
			t.Errorf("nats server not ready for connection after %d seconds", 4)
			os.Exit(1)
		}
		serverReady <- true
		<-serverDone

		t.Log("Closing nats server")
		ns.Shutdown()
		wg.Done()
	}()

	<-serverReady
	t.Log("nats server started")

	r, err := NewNatsReceiver("testreceiver", natsReceiverTestConfig)
	if err != nil {
		t.Errorf("failed to start nats receiver with config '%s': %s", string(httpReceiverTestConfig), err.Error())
		return
	}
	r.SetSink(sink)
	t.Log("Starting nats receiver")
	r.Start()

	msgs := gen_messages(numMessage)

	wg.Add(1)
	go func() {
		t.Logf("connecting nats client to %s", uri)
		c, err := nats.Connect(uri, nil)
		if err != nil {
			t.Errorf("failed to connect to nats server %s: %s", uri, err.Error())
			os.Exit(1)
		}
		for i, m := range msgs {
			ilp := m.ToLineProtocol(nil)
			t.Logf("publishing to subject test: %s", ilp)
			err := c.Publish("test", []byte(ilp))
			if err != nil {
				t.Errorf("failed sending '%s': %s", ilp, err.Error())
			}
			c.Flush()
			if i == 0 {
				sendDone <- true
			}
		}

		t.Log("Closing nats client")
		c.Close()
		wg.Done()
	}()

	recvm := make([]lp.CCMessage, 0, numMessage)
	<-sendDone
	for i := 0; i < numMessage; i++ {
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

	serverDone <- true
	wg.Wait()

}
