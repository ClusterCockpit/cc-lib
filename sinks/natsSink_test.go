package sinks

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	server "github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
)

var testNatsConfig = NatsSinkConfig{
	Host:       "localhost",
	Port:       "4222",
	FlushDelay: "1s",
	Precision:  "ns",
	Subject:    "testsubject",
}

func TestNatsSink(t *testing.T) {
	receivedMessages := make([]string, 0)
	jsonConfig, err := json.Marshal(testNatsConfig)
	if err != nil {
		t.Errorf("failed to marshal configuration: %s", err.Error())
		return
	}

	opts := &server.Options{
		Host: "localhost",
		Port: 4222,
	}

	uri := fmt.Sprintf("nats://%s:%d", opts.Host, opts.Port)
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

	sub, err := c.Subscribe("testsubject", func(msg *nats.Msg) {
		if err == nil {
			t.Log(string(msg.Data))
			lines := strings.Split(string(msg.Data), "\n")
			for _, l := range lines {
				if len(l) > 0 {
					receivedMessages = append(receivedMessages, l)
				}
			}
		}
	})

	t.Logf("setup nats sink to %s:%s subject %s", testNatsConfig.Host, testNatsConfig.Port, testNatsConfig.Subject)
	s, err := NewNatsSink("testsink", jsonConfig)
	if err != nil {
		t.Errorf("failed to setup nats sink: %s", err.Error())
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
	time.Sleep(time.Second)

	t.Log("shutdown nats sink")
	s.Close()

	t.Log("shutdown nats client")
	sub.Unsubscribe()
	c.Close()

	t.Log("shutdown nats server")
	ns.Shutdown()

	if len(msgs) != len(receivedMessages) {
		t.Error("not all messages sent and received")
		return
	}

	for i, m := range msgs {
		ms := m.ToLineProtocol(nil)
		ms = strings.Trim(ms, "\n")
		if ms != receivedMessages[i] {
			t.Errorf("message %d invalid: '%s' vs '%s'", i, ms, receivedMessages[i])
		}
	}
}
