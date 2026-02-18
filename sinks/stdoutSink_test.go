package sinks

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type testStdoutConfig struct {
	defaultSinkConfig
	Output string `json:"output_file,omitempty"`
}

func TestStdoutSink(t *testing.T) {
	receivedStdoutMessages := make([]string, 0)
	f, err := os.CreateTemp("", "tmpfile-")
	if err != nil {
		t.Errorf("failed to create temporary file: %s", err.Error())
		return
	}
	defer f.Close()
	defer os.Remove(f.Name())
	t.Logf("using temporary file %s", f.Name())

	stdoutSinkConfig := testStdoutConfig{
		Output: f.Name(),
	}
	jsonConfig, err := json.Marshal(stdoutSinkConfig)
	if err != nil {
		t.Errorf("failed to marshal configuration: %s", err.Error())
		return
	}
	t.Logf("setup stdout sink to %s", f.Name())
	s, err := NewStdoutSink("testsink", jsonConfig)
	if err != nil {
		t.Errorf("failed to setup stdout sink: %s", err.Error())
	}

	msgs, _ := gen_messages(10)
	t.Logf("writing %d messages to stdout sink", len(msgs))
	for _, m := range msgs {
		t.Log(m.String())
		s.Write(m)
	}
	t.Log("flushing stdout sink")
	s.Flush()

	t.Logf("reading messages from %s", f.Name())
	data, err := os.ReadFile(f.Name())
	lines := strings.SplitSeq(string(data), "\n")
	for l := range lines {
		if len(l) > 0 {
			receivedStdoutMessages = append(receivedStdoutMessages, l)
		}
	}
	t.Log("shutdown stdout sink")
	s.Close()

	if len(msgs) != len(receivedStdoutMessages) {
		t.Error("not all messages sent and received")
		return
	}

	for i, m := range msgs {
		ms := m.ToLineProtocol(nil)
		ms = strings.Trim(ms, "\n")
		if ms != receivedStdoutMessages[i] {
			t.Errorf("message %d invalid: '%s' vs '%s'", i, ms, receivedStdoutMessages[i])
		}
	}
}
