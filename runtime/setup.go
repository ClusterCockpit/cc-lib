// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package runtime provides utilities for runtime environment setup and systemd integration
// for ClusterCockpit applications.
//
// This package enables applications to:
//   - Load environment variables from .env configuration files
//   - Drop root privileges to unprivileged users for enhanced security
//   - Integrate with systemd's service lifecycle notification protocol
//
// # Environment File Loading
//
// LoadEnv reads .env files and adds all variable definitions to the process environment.
// It supports a simple subset of .env file syntax:
//
//	# Comments start with #
//	SIMPLE_VAR=value
//	export EXPORTED_VAR=value
//	QUOTED_VAR="value with spaces"
//	ESCAPED_VAR="line1\nline2\ttabbed"
//
// Supported escape sequences in quoted strings: \n (newline), \r (carriage return),
// \t (tab), \" (quote). Comments are only allowed at the start of lines.
//
// # Privilege Dropping
//
// DropPrivileges allows services that start as root to permanently drop to unprivileged
// users for security. This is critical for services that need root access initially
// (e.g., to bind to ports < 1024) but should run with minimal privileges afterward.
//
// The Go runtime ensures all threads (not just the calling thread) execute the
// underlying syscalls, making this safe for multi-threaded applications.
//
// Security best practice: Always drop privileges as early as possible after completing
// privileged operations.
//
// # Systemd Integration
//
// SystemdNotify enables proper integration with systemd's notification protocol,
// allowing services to:
//   - Signal when they're ready to accept requests (Type=notify services)
//   - Update status messages visible in systemctl status
//   - Implement proper service lifecycle management
//
// The function safely handles non-systemd environments by checking for the
// NOTIFY_SOCKET environment variable.
//
// See: https://www.freedesktop.org/software/systemd/man/sd_notify.html
//
// # Thread Safety
//
// All functions in this package are safe to call from multiple goroutines.
// However, DropPrivileges should only be called once during application startup.

package runtime

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

// LoadEnv reads a .env file and adds all variable definitions to the process environment.
//
// The function supports a simple .env file format:
//   - Comments: Lines starting with # are ignored
//   - Empty lines: Blank lines are ignored
//   - Export prefix: "export VAR=value" is supported (export is stripped)
//   - Quoted values: Double-quoted strings support escape sequences
//   - Key-value pairs: Must be in the format KEY=VALUE
//
// Escape sequences in quoted strings:
//   - \n: newline
//   - \r: carriage return
//   - \t: tab
//   - \": double quote
//
// Limitations:
//   - Comments are only allowed at the start of lines (not inline)
//   - Only double quotes are supported (not single quotes)
//   - No variable expansion or substitution
//   - No multi-line values
//
// Parameters:
//   - file: Path to the .env file to load
//
// Returns:
//   - error: nil on success, error if file cannot be read or contains invalid syntax
//
// Examples:
//
//	// Load .env file
//	if err := runtimeEnv.LoadEnv("./.env"); err != nil && !os.IsNotExist(err) {
//	    log.Fatalf("Failed to load .env: %v", err)
//	}
//
//	// It's safe to ignore file-not-found errors
//	_ = runtimeEnv.LoadEnv("./.env") // Optional config file
//
// Example .env file:
//
//	# Database configuration
//	DB_HOST=localhost
//	DB_PORT=5432
//	export DB_NAME=myapp
//	DB_PASSWORD="secret with spaces"
//	LOG_FORMAT="timestamp\tlevel\tmessage\n"
func LoadEnv(file string) error {
	f, err := os.Open(file)
	if err != nil {
		cclog.Errorf("Error while opening .env file: %v", err)
		return err
	}

	defer f.Close()
	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}

		if strings.Contains(line, "#") {
			return errors.New("'#' are only supported at the start of a line")
		}

		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("unsupported line: %#v", line)
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if strings.HasPrefix(val, "\"") {
			if !strings.HasSuffix(val, "\"") {
				return fmt.Errorf("unsupported line: %#v", line)
			}

			runes := []rune(val[1 : len(val)-1])
			sb := strings.Builder{}
			for i := 0; i < len(runes); i++ {
				if runes[i] == '\\' {
					i++
					if i >= len(runes) {
						return fmt.Errorf("invalid escape sequence at end of string: %#v", line)
					}
					switch runes[i] {
					case 'n':
						sb.WriteRune('\n')
					case 'r':
						sb.WriteRune('\r')
					case 't':
						sb.WriteRune('\t')
					case '"':
						sb.WriteRune('"')
					default:
						return fmt.Errorf("unsupported escape sequence in quoted string: backslash %#v", runes[i])
					}
					continue
				}
				sb.WriteRune(runes[i])
			}

			val = sb.String()
		}

		os.Setenv(key, val)
	}

	return s.Err()
}

// DropPrivileges permanently changes the process user and group to the specified
// unprivileged account for enhanced security.
//
// This function is typically used by services that start as root but should run with
// minimal privileges. The Go runtime ensures all OS threads execute the underlying
// syscalls, making this safe for multi-threaded applications.
//
// The function drops privileges in the correct order:
//  1. Set group ID first (requires root privileges)
//  2. Set user ID second (after this, root privileges are permanently lost)
//
// Security notes:
//   - This operation is permanent and irreversible within the process
//   - Always verify the user/group exist before calling
//   - Call this as early as possible after completing privileged operations
//   - Both parameters are optional; empty strings skip that operation
//
// Parameters:
//   - username: Username to switch to (empty string skips user change)
//   - group: Group name to switch to (empty string skips group change)
//
// Returns:
//   - error: nil on success, error if user/group lookup fails or syscall fails
//
// Examples:
//
//	// Drop to dedicated service user
//	if err := runtimeEnv.DropPrivileges("ccuser", "ccgroup"); err != nil {
//	    log.Fatalf("Failed to drop privileges: %v", err)
//	}
//
//	// Only change user, keep current group
//	if err := runtimeEnv.DropPrivileges("nobody", ""); err != nil {
//	    log.Fatalf("Failed to drop privileges: %v", err)
//	}
//
//	// Typical usage pattern
//	func main() {
//	    // Perform privileged operations (bind to port 80, etc.)
//	    if err := startServer(); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Drop privileges before handling requests
//	    if err := runtimeEnv.DropPrivileges("www-data", "www-data"); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Now running as unprivileged user
//	    serveRequests()
//	}
func DropPrivileges(username string, group string) error {
	if group != "" {
		g, err := user.LookupGroup(group)
		if err != nil {
			cclog.Warnf("Error while looking up group '%s': %v", group, err)
			return err
		}

		gid, err := strconv.Atoi(g.Gid)
		if err != nil {
			cclog.Warnf("Error while parsing group GID '%s': %v", g.Gid, err)
			return err
		}

		if err := syscall.Setgid(gid); err != nil {
			cclog.Warnf("Error while setting gid %d: %v", gid, err)
			return err
		}
	}

	if username != "" {
		u, err := user.Lookup(username)
		if err != nil {
			cclog.Warnf("Error while looking up user '%s': %v", username, err)
			return err
		}

		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			cclog.Warnf("Error while parsing user UID '%s': %v", u.Uid, err)
			return err
		}

		if err := syscall.Setuid(uid); err != nil {
			cclog.Warnf("Error while setting uid %d: %v", uid, err)
			return err
		}
	}

	return nil
}

// SystemdNotify sends service status notifications to systemd when running as a
// systemd service (Type=notify).
//
// This function implements the systemd notification protocol, allowing services to:
//   - Signal readiness to accept requests (ready=true)
//   - Update status messages visible in "systemctl status"
//   - Implement proper service lifecycle management
//
// The function safely handles non-systemd environments by checking for the
// NOTIFY_SOCKET environment variable. If not running under systemd, it returns
// immediately without error.
//
// Errors from systemd-notify are intentionally ignored as there's limited
// recovery action and the service should continue running.
//
// Parameters:
//   - ready: If true, signals the service is ready (sends --ready)
//   - status: Status message to display (empty string skips status update)
//
// Examples:
//
//	// Signal service is ready
//	runtimeEnv.SystemdNotify(true, "Ready to accept connections")
//
//	// Update status without signaling ready
//	runtimeEnv.SystemdNotify(false, "Processing 1000 requests/sec")
//
//	// Shutdown notification
//	runtimeEnv.SystemdNotify(false, "Shutting down gracefully")
//
//	// Typical usage in main()
//	func main() {
//	    // Initialize application
//	    if err := initialize(); err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Signal systemd we're ready
//	    runtimeEnv.SystemdNotify(true, "Running")
//
//	    // Run service
//	    serve()
//	}
//
// Systemd service file example:
//
//	[Service]
//	Type=notify
//	ExecStart=/usr/bin/myservice
//	NotifyAccess=main
//
// See: https://www.freedesktop.org/software/systemd/man/sd_notify.html
func SystemdNotify(ready bool, status string) {
	if os.Getenv("NOTIFY_SOCKET") == "" {
		// Not started using systemd
		return
	}

	args := []string{fmt.Sprintf("--pid=%d", os.Getpid())}
	if ready {
		args = append(args, "--ready")
	}

	if status != "" {
		args = append(args, fmt.Sprintf("--status=%s", status))
	}

	cmd := exec.Command("systemd-notify", args...)
	cmd.Run() // errors ignored on purpose, there is not much to do anyways.
}
