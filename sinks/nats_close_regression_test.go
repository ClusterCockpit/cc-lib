// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package sinks

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNatsSinkCloseDeliversBufferedMessages(t *testing.T) {
	sink, sub := newProcessingTestNatsSink(t)
	messages, err := gen_messages(3)
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range messages {
		if err = sink.Write(msg); err != nil {
			t.Fatal(err)
		}
	}
	sink.Close()
	msg, err := sub.NextMsg(time.Second)
	if err != nil {
		t.Fatalf("Close lost accepted buffered messages: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(msg.Data), "\n"), "\n")
	if len(lines) != len(messages) {
		t.Fatalf("got %d messages, want %d", len(lines), len(messages))
	}
	for i, input := range messages {
		want := strings.TrimSuffix(input.ToLineProtocol(nil), "\n")
		// This fixture configures second precision; ToLineProtocol uses nanoseconds.
		separator := strings.LastIndexByte(want, ' ')
		want = want[:separator+1] + strconv.FormatInt(input.Time().Unix(), 10)
		if lines[i] != want {
			t.Fatalf("message %d: got %q, want %q", i, lines[i], want)
		}
	}
}
