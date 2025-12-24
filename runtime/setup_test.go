// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
		checkVars   map[string]string
	}{
		{
			name: "simple variables",
			content: `SIMPLE=value
ANOTHER=test123`,
			wantErr: false,
			checkVars: map[string]string{
				"SIMPLE":  "value",
				"ANOTHER": "test123",
			},
		},
		{
			name: "export prefix",
			content: `export EXPORTED=value
export ANOTHER_EXPORT=test`,
			wantErr: false,
			checkVars: map[string]string{
				"EXPORTED":       "value",
				"ANOTHER_EXPORT": "test",
			},
		},
		{
			name: "quoted strings",
			content: `QUOTED="value with spaces"
EMPTY=""
SINGLE_WORD="word"`,
			wantErr: false,
			checkVars: map[string]string{
				"QUOTED":      "value with spaces",
				"EMPTY":       "",
				"SINGLE_WORD": "word",
			},
		},
		{
			name: "escape sequences",
			content: `NEWLINE="line1\nline2"
TAB="col1\tcol2"
QUOTE="say \"hello\""
RETURN="text\rmore"`,
			wantErr: false,
			checkVars: map[string]string{
				"NEWLINE": "line1\nline2",
				"TAB":     "col1\tcol2",
				"QUOTE":   "say \"hello\"",
				"RETURN":  "text\rmore",
			},
		},
		{
			name: "comments and empty lines",
			content: `# This is a comment
VAR1=value1

# Another comment
VAR2=value2

`,
			wantErr: false,
			checkVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
		{
			name: "mixed formats",
			content: `# Configuration
SIMPLE=value
export EXPORTED=another
QUOTED="with spaces"
ESCAPED="tab\there"`,
			wantErr: false,
			checkVars: map[string]string{
				"SIMPLE":   "value",
				"EXPORTED": "another",
				"QUOTED":   "with spaces",
				"ESCAPED":  "tab\there",
			},
		},
		{
			name:        "inline comment - not supported",
			content:     `VAR=value # inline comment`,
			wantErr:     true,
			errContains: "'#' are only supported at the start of a line",
		},
		{
			name:        "missing equals",
			content:     `INVALID_LINE`,
			wantErr:     true,
			errContains: "unsupported line",
		},
		{
			name:        "unclosed quote",
			content:     `UNCLOSED="missing end quote`,
			wantErr:     true,
			errContains: "unsupported line",
		},
		{
			name:        "invalid escape sequence",
			content:     `INVALID="bad\xescape"`,
			wantErr:     true,
			errContains: "unsupported escape sequence",
		},
		{
			name:        "unclosed quote",
			content:     `INVALID="trailing\`,
			wantErr:     true,
			errContains: "unsupported line",
		},
		{
			name: "special characters in values",
			content: `URL=https://example.com:8080/path?query=value
EMAIL=user@example.com
PATH=/usr/local/bin:/usr/bin`,
			wantErr: false,
			checkVars: map[string]string{
				"URL":   "https://example.com:8080/path?query=value",
				"EMAIL": "user@example.com",
				"PATH":  "/usr/local/bin:/usr/bin",
			},
		},
		{
			name: "whitespace handling",
			content: `  TRIMMED  =  value  
QUOTED_WS="  preserved  "`,
			wantErr: false,
			checkVars: map[string]string{
				"TRIMMED":   "value",
				"QUOTED_WS": "  preserved  ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, ".env")

			if err := os.WriteFile(envFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("Failed to create temp .env file: %v", err)
			}

			// Clear environment variables we're testing
			for key := range tt.checkVars {
				os.Unsetenv(key)
			}

			// Test LoadEnv
			err := LoadEnv(envFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadEnv() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("LoadEnv() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadEnv() unexpected error: %v", err)
				return
			}

			// Check environment variables
			for key, expected := range tt.checkVars {
				actual := os.Getenv(key)
				if actual != expected {
					t.Errorf("Environment variable %s = %q, want %q", key, actual, expected)
				}
			}
		})
	}
}

func TestLoadEnv_FileNotFound(t *testing.T) {
	err := LoadEnv("/nonexistent/path/.env")
	if err == nil {
		t.Error("LoadEnv() expected error for nonexistent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("LoadEnv() error should be os.IsNotExist, got: %v", err)
	}
}

func TestDropPrivileges_InvalidUser(t *testing.T) {
	// Skip if running as root (would actually succeed)
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Test with non-existent user
	err := DropPrivileges("nonexistent_user_12345", "")
	if err == nil {
		t.Error("DropPrivileges() expected error for invalid user")
	}
}

func TestDropPrivileges_InvalidGroup(t *testing.T) {
	// Skip if running as root (would actually succeed)
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Test with non-existent group
	err := DropPrivileges("", "nonexistent_group_12345")
	if err == nil {
		t.Error("DropPrivileges() expected error for invalid group")
	}
}

func TestDropPrivileges_EmptyBoth(t *testing.T) {
	// Should succeed with no-op when both are empty
	err := DropPrivileges("", "")
	if err != nil {
		t.Errorf("DropPrivileges(\"\", \"\") should succeed, got error: %v", err)
	}
}

func TestSystemdNotify_NoSocket(t *testing.T) {
	// Clear NOTIFY_SOCKET to simulate non-systemd environment
	oldValue := os.Getenv("NOTIFY_SOCKET")
	os.Unsetenv("NOTIFY_SOCKET")
	defer func() {
		if oldValue != "" {
			os.Setenv("NOTIFY_SOCKET", oldValue)
		}
	}()

	// Should not panic or error
	SystemdNotify(true, "test")
	SystemdNotify(false, "test")
	SystemdNotify(true, "")
}

func TestSystemdNotify_WithSocket(t *testing.T) {
	// Set NOTIFY_SOCKET to a dummy path
	// (systemd-notify will fail but that's expected and ignored)
	oldValue := os.Getenv("NOTIFY_SOCKET")
	os.Setenv("NOTIFY_SOCKET", "/tmp/test-notify.sock")
	defer func() {
		if oldValue != "" {
			os.Setenv("NOTIFY_SOCKET", oldValue)
		} else {
			os.Unsetenv("NOTIFY_SOCKET")
		}
	}()

	// Should not panic even if systemd-notify fails
	SystemdNotify(true, "Ready")
	SystemdNotify(false, "Status update")
	SystemdNotify(true, "")
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
