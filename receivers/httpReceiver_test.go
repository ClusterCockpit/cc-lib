package receivers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
)

var httpReceiverTestConfig json.RawMessage = json.RawMessage(`{
	"type": "http",
	"address" : "localhost",
	"port" : "8082",
	"path" : "/write"
}`)

func gen_messages(numMessages int) []lp.CCMessage {
	out := make([]lp.CCMessage, 0, numMessages)
	for i := 0; i < numMessages; i++ {
		x, err := lp.NewMetric("testmetric", map[string]string{"type": "node"}, nil, 42.7*(0.6*float64(i)), time.Now())
		if err == nil {
			out = append(out, x)
		}
	}
	return out
}

func TestHttpReceiver(t *testing.T) {
	numMessage := 10

	sink := make(chan lp.CCMessage, numMessage)
	r, err := NewHttpReceiver("testreceiver", httpReceiverTestConfig)
	if err != nil {
		t.Errorf("failed to start http receiver with config '%s': %s", string(httpReceiverTestConfig), err.Error())
		return
	}
	r.SetSink(sink)
	t.Log("Starting http receiver")
	r.Start()
	time.Sleep(time.Second)

	msgs := gen_messages(numMessage)
	for _, m := range msgs {
		ilp := m.ToLineProtocol(nil)
		mr := strings.NewReader(ilp)
		_, err := http.Post("http://localhost:8082/write", "application/text", mr)
		if err != nil {
			t.Errorf("failed sending '%s': %s", ilp, err.Error())
		}
	}

	recvm := make([]lp.CCMessage, 0, numMessage)
	for i := 0; i < numMessage && len(sink) > 0; i++ {
		recvm = append(recvm, <-sink)
	}
	if len(recvm) != len(msgs) {
		t.Errorf("received only %d metrics", len(recvm))
	}
	for i := 0; i < numMessage; i++ {
		if msgs[i].ToLineProtocol(nil) != recvm[i].ToLineProtocol(nil) {
			t.Errorf("metrics do no match '%s' vs '%s'", msgs[i].ToLineProtocol(nil), recvm[i].ToLineProtocol(nil))
		}
	}

	t.Log("Closing http receiver")
	r.Close()
}
