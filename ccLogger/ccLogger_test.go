// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package cclogger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestInit tests the initialization of cclogger with various log levels
func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		logdate bool
		want    string
	}{
		{"info level no date", "info", false, "info"},
		{"debug level no date", "debug", false, "debug"},
		{"warn level no date", "warn", false, "warn"},
		{"error level no date", "err", false, "err"},
		{"fatal level no date", "fatal", false, "fatal"},
		{"crit level no date", "crit", false, "crit"},
		{"info level with date", "info", true, "info"},
		{"invalid level", "invalid", false, "debug"}, // Should default to debug
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.level, tt.logdate)
			got := Loglevel()
			if got != tt.want {
				t.Errorf("Loglevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLogLevelFiltering tests that log levels are properly filtered
func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectDebug bool
		expectInfo  bool
		expectWarn  bool
		expectError bool
		expectCrit  bool
	}{
		{"debug shows all", "debug", true, true, true, true, true},
		{"info filters debug", "info", false, true, true, true, true},
		{"warn filters info and debug", "warn", false, false, true, true, true},
		{"err filters warn, info, debug", "err", false, false, false, true, true},
		{"crit filters all except crit", "crit", false, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var debugBuf, infoBuf, warnBuf, errBuf, critBuf bytes.Buffer

			// Reset writers to capture output
			DebugWriter = &debugBuf
			InfoWriter = &infoBuf
			WarnWriter = &warnBuf
			ErrWriter = &errBuf
			CritWriter = &critBuf

			Init(tt.level, false)

			// Check if writers are set to io.Discard or actual buffer
			debugIsDiscard := DebugWriter == io.Discard
			infoIsDiscard := InfoWriter == io.Discard
			warnIsDiscard := WarnWriter == io.Discard
			errIsDiscard := ErrWriter == io.Discard
			critIsDiscard := CritWriter == io.Discard

			if debugIsDiscard == tt.expectDebug {
				t.Errorf("Debug: got discarded=%v, want active=%v", debugIsDiscard, tt.expectDebug)
			}
			if infoIsDiscard == tt.expectInfo {
				t.Errorf("Info: got discarded=%v, want active=%v", infoIsDiscard, tt.expectInfo)
			}
			if warnIsDiscard == tt.expectWarn {
				t.Errorf("Warn: got discarded=%v, want active=%v", warnIsDiscard, tt.expectWarn)
			}
			if errIsDiscard == tt.expectError {
				t.Errorf("Error: got discarded=%v, want active=%v", errIsDiscard, tt.expectError)
			}
			if critIsDiscard == tt.expectCrit {
				t.Errorf("Crit: got discarded=%v, want active=%v", critIsDiscard, tt.expectCrit)
			}

			// Reset to stderr for other tests
			DebugWriter = os.Stderr
			InfoWriter = os.Stderr
			WarnWriter = os.Stderr
			ErrWriter = os.Stderr
			CritWriter = os.Stderr
		})
	}
}

// TestLogOutput tests that log functions produce output
func TestLogOutput(t *testing.T) {
	var buf bytes.Buffer

	// Set all writers to the buffer
	DebugWriter = &buf
	InfoWriter = &buf
	WarnWriter = &buf
	ErrWriter = &buf
	CritWriter = &buf

	Init("debug", false)

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		Debug("test debug message")
		if !strings.Contains(buf.String(), "test debug message") {
			t.Errorf("Debug() output missing expected message, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "[DEBUG]") {
			t.Errorf("Debug() output missing debug prefix, got: %s", buf.String())
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		Info("test info message")
		if !strings.Contains(buf.String(), "test info message") {
			t.Errorf("Info() output missing expected message, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "[INFO]") {
			t.Errorf("Info() output missing info prefix, got: %s", buf.String())
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		Warn("test warn message")
		if !strings.Contains(buf.String(), "test warn message") {
			t.Errorf("Warn() output missing expected message, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "[WARNING]") {
			t.Errorf("Warn() output missing warning prefix, got: %s", buf.String())
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		Error("test error message")
		if !strings.Contains(buf.String(), "test error message") {
			t.Errorf("Error() output missing expected message, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "[ERROR]") {
			t.Errorf("Error() output missing error prefix, got: %s", buf.String())
		}
	})

	// Reset to stderr
	DebugWriter = os.Stderr
	InfoWriter = os.Stderr
	WarnWriter = os.Stderr
	ErrWriter = os.Stderr
	CritWriter = os.Stderr
}

// TestFormattedOutput tests formatted log functions
func TestFormattedOutput(t *testing.T) {
	var buf bytes.Buffer

	DebugWriter = &buf
	InfoWriter = &buf
	WarnWriter = &buf
	ErrWriter = &buf
	CritWriter = &buf

	Init("debug", false)

	t.Run("Debugf", func(t *testing.T) {
		buf.Reset()
		Debugf("formatted %s %d", "message", 42)
		if !strings.Contains(buf.String(), "formatted message 42") {
			t.Errorf("Debugf() output incorrect, got: %s", buf.String())
		}
	})

	t.Run("Infof", func(t *testing.T) {
		buf.Reset()
		Infof("formatted %s %d", "message", 42)
		if !strings.Contains(buf.String(), "formatted message 42") {
			t.Errorf("Infof() output incorrect, got: %s", buf.String())
		}
	})

	t.Run("Warnf", func(t *testing.T) {
		buf.Reset()
		Warnf("formatted %s %d", "message", 42)
		if !strings.Contains(buf.String(), "formatted message 42") {
			t.Errorf("Warnf() output incorrect, got: %s", buf.String())
		}
	})

	t.Run("Errorf", func(t *testing.T) {
		buf.Reset()
		Errorf("formatted %s %d", "message", 42)
		if !strings.Contains(buf.String(), "formatted message 42") {
			t.Errorf("Errorf() output incorrect, got: %s", buf.String())
		}
	})

	// Reset to stderr
	DebugWriter = os.Stderr
	InfoWriter = os.Stderr
	WarnWriter = os.Stderr
	ErrWriter = os.Stderr
	CritWriter = os.Stderr
}

// TestComponentLogging tests component-specific logging functions
func TestComponentLogging(t *testing.T) {
	var buf bytes.Buffer

	DebugWriter = &buf
	InfoWriter = &buf
	WarnWriter = &buf
	ErrWriter = &buf

	Init("debug", false)

	t.Run("ComponentDebug", func(t *testing.T) {
		buf.Reset()
		ComponentDebug("testcomp", "debug message")
		output := buf.String()
		if !strings.Contains(output, "[testcomp]") {
			t.Errorf("ComponentDebug() missing component prefix, got: %s", output)
		}
		if !strings.Contains(output, "debug message") {
			t.Errorf("ComponentDebug() missing message, got: %s", output)
		}
	})

	t.Run("ComponentInfo", func(t *testing.T) {
		buf.Reset()
		ComponentInfo("testcomp", "info message")
		output := buf.String()
		if !strings.Contains(output, "[testcomp]") {
			t.Errorf("ComponentInfo() missing component prefix, got: %s", output)
		}
		if !strings.Contains(output, "info message") {
			t.Errorf("ComponentInfo() missing message, got: %s", output)
		}
	})

	t.Run("ComponentWarn", func(t *testing.T) {
		buf.Reset()
		ComponentWarn("testcomp", "warn message")
		output := buf.String()
		if !strings.Contains(output, "[testcomp]") {
			t.Errorf("ComponentWarn() missing component prefix, got: %s", output)
		}
		if !strings.Contains(output, "warn message") {
			t.Errorf("ComponentWarn() missing message, got: %s", output)
		}
	})

	t.Run("ComponentError", func(t *testing.T) {
		buf.Reset()
		ComponentError("testcomp", "error message")
		output := buf.String()
		if !strings.Contains(output, "[testcomp]") {
			t.Errorf("ComponentError() missing component prefix, got: %s", output)
		}
		if !strings.Contains(output, "error message") {
			t.Errorf("ComponentError() missing message, got: %s", output)
		}
	})

	t.Run("ComponentMultipleArgs", func(t *testing.T) {
		buf.Reset()
		ComponentInfo("testcomp", "message", "part", 2)
		output := buf.String()
		if !strings.Contains(output, "[testcomp]") {
			t.Errorf("ComponentInfo() missing component prefix with multiple args, got: %s", output)
		}
		if !strings.Contains(output, "message") || !strings.Contains(output, "part") || !strings.Contains(output, "2") {
			t.Errorf("ComponentInfo() missing message parts with multiple args, got: %s", output)
		}
	})

	// Reset to stderr
	DebugWriter = os.Stderr
	InfoWriter = os.Stderr
	WarnWriter = os.Stderr
	ErrWriter = os.Stderr
}

// TestSetOutputFile tests file output functionality
func TestSetOutputFile(t *testing.T) {
	dir := t.TempDir()
	logfile := dir + "/output.log"

	// Initialize logger
	Init("debug", false)

	// Set output to file for warn level (should affect warn, info, debug)
	SetOutputFile("warn", logfile)

	// Log at different levels
	Debug("debug message to file")
	Info("info message to file")
	Warn("warn message to file")

	// Read the file
	content, err := os.ReadFile(logfile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)

	// Check that all three messages are in the file
	if !strings.Contains(output, "debug message to file") {
		t.Errorf("Debug message not written to file")
	}
	if !strings.Contains(output, "info message to file") {
		t.Errorf("Info message not written to file")
	}
	if !strings.Contains(output, "warn message to file") {
		t.Errorf("Warn message not written to file")
	}
}

// TestSetOutputFileInvalidLevel tests SetOutputFile with invalid log level
func TestSetOutputFileInvalidLevel(t *testing.T) {
	dir := t.TempDir()
	logfile := dir + "/output.log"

	var buf bytes.Buffer
	ErrWriter = &buf

	Init("debug", false)

	// This should log an error
	SetOutputFile("invalid", logfile)

	output := buf.String()
	if !strings.Contains(output, "invalid loglevel") {
		t.Errorf("Expected error message for invalid level, got: %s", output)
	}

	// Reset
	ErrWriter = os.Stderr
}

// TestConcurrentLogging tests thread safety of logging functions
func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	InfoWriter = &buf

	Init("info", false)

	const goroutines = 10
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range messagesPerGoroutine {
				Infof("goroutine %d message %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Just verify we didn't panic and wrote something
	if buf.Len() == 0 {
		t.Error("No output from concurrent logging")
	}

	// Reset
	InfoWriter = os.Stderr
}

// TestEdgeCases tests edge cases
func TestEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	InfoWriter = &buf

	Init("info", false)

	t.Run("EmptyMessage", func(t *testing.T) {
		buf.Reset()
		Info()
		// Should not panic
	})

	t.Run("EmptyFormatString", func(t *testing.T) {
		buf.Reset()
		Infof("")
		// Should not panic
	})

	t.Run("LargeMessage", func(t *testing.T) {
		buf.Reset()
		largeMsg := strings.Repeat("x", 10000)
		Info(largeMsg)
		if !strings.Contains(buf.String(), largeMsg) {
			t.Error("Large message not logged correctly")
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		buf.Reset()
		Info("message with\nnewlines\tand\ttabs")
		output := buf.String()
		if !strings.Contains(output, "newlines") || !strings.Contains(output, "tabs") {
			t.Errorf("Special characters not handled correctly, got: %s", output)
		}
	})

	// Reset
	InfoWriter = os.Stderr
}

// TestPanic tests the Panic function
func TestPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Panic() did not panic")
		} else {
			if !strings.Contains(r.(string), "panic test") {
				t.Errorf("Panic() message incorrect, got: %v", r)
			}
		}
	}()

	Panic("panic test")
}

// TestPanicf tests the Panicf function
func TestPanicf(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Panicf() did not panic")
		} else {
			if !strings.Contains(r.(string), "panic test 42") {
				t.Errorf("Panicf() message incorrect, got: %v", r)
			}
		}
	}()

	Panicf("panic test %d", 42)
}

// TestSystemdPrefixes tests that systemd prefixes are present
func TestSystemdPrefixes(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(string)
		prefix  string
		level   string
	}{
		{"Debug", func(s string) { Debug(s) }, "<7>", "DEBUG"},
		{"Info", func(s string) { Info(s) }, "<6>", "INFO"},
		{"Warn", func(s string) { Warn(s) }, "<4>", "WARNING"},
		{"Error", func(s string) { Error(s) }, "<3>", "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			DebugWriter = &buf
			InfoWriter = &buf
			WarnWriter = &buf
			ErrWriter = &buf

			Init("debug", false)

			tt.logFunc("test")
			output := buf.String()

			if !strings.Contains(output, tt.prefix) {
				t.Errorf("%s missing systemd prefix %s, got: %s", tt.name, tt.prefix, output)
			}
			if !strings.Contains(output, tt.level) {
				t.Errorf("%s missing level marker %s, got: %s", tt.name, tt.level, output)
			}
		})
	}

	// Reset
	DebugWriter = os.Stderr
	InfoWriter = os.Stderr
	WarnWriter = os.Stderr
	ErrWriter = os.Stderr
}

// TestLoglevelGetter tests the Loglevel() function
func TestLoglevelGetter(t *testing.T) {
	levels := []string{"debug", "info", "warn", "err", "crit"}

	for _, lvl := range levels {
		Init(lvl, false)
		got := Loglevel()
		if got != lvl {
			t.Errorf("After Init(%q), Loglevel() = %q, want %q", lvl, got, lvl)
		}
	}
}
