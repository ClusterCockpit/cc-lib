// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cclogger implements a simple log wrapper for the standard log package.
//
// cclogger provides a simple way of logging with different levels (debug, info, warn, error, critical).
// It integrates with systemd's journaling system by using standard systemd log level prefixes.
//
// # Log Levels
//
// The package supports the following log levels in order of increasing severity:
//   - debug: Detailed information for development and troubleshooting
//   - info: General informational messages about application progress
//   - warn: Warning messages for important but non-critical issues
//   - err/fatal: Error messages for failures that may allow continued execution
//   - crit: Critical errors that typically result in program termination
//
// # Basic Usage
//
// Initialize the logger with a log level and optional timestamp:
//
//	cclogger.Init("info", false) // info level, no timestamps (systemd adds them)
//
// Log messages using level-specific functions:
//
//	cclogger.Debug("Detailed debug information")
//	cclogger.Info("Application started")
//	cclogger.Warn("Configuration value missing, using default")
//	cclogger.Error("Failed to connect to database")
//	cclogger.Fatal("Critical error, exiting") // exits with code 1
//
// Use formatted variants for structured output:
//
//	cclogger.Infof("Processing %d items", count)
//	cclogger.Errorf("Failed to open file: %v", err)
//
// # Component Logging
//
// For component-specific logging, use the Component* variants:
//
//	cclogger.ComponentInfo("scheduler", "Job submitted successfully")
//	cclogger.ComponentError("database", "Connection pool exhausted")
//
// # Thread Safety
//
// All logging functions are thread-safe as they use the standard library's log.Logger,
// which is safe for concurrent use.
//
// # Systemd Integration
//
// Log messages use systemd's priority prefixes (<7> for debug, <6> for info, etc.)
// which allows systemd-journald to properly categorize log entries. When running
// under systemd, timestamps can be omitted (logdate=false) as journald adds them.
package cclogger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// Package-level writers for each log level. These can be configured to direct
// output to different destinations. By default, all output goes to stderr.
// When Init() is called with a log level, writers for levels below the threshold
// are set to io.Discard to suppress output.
var (
	// DebugWriter is the output destination for debug-level logs
	DebugWriter io.Writer = os.Stderr
	// InfoWriter is the output destination for info-level logs
	InfoWriter io.Writer = os.Stderr
	// WarnWriter is the output destination for warning-level logs
	WarnWriter io.Writer = os.Stderr
	// ErrWriter is the output destination for error-level logs
	ErrWriter io.Writer = os.Stderr
	// CritWriter is the output destination for critical-level logs
	CritWriter io.Writer = os.Stderr
)

// Log level prefixes using systemd priority levels.
// The numeric prefix (<N>) corresponds to syslog/systemd severity levels:
//
//	<7> = Debug, <6> = Informational, <4> = Warning, <3> = Error, <2> = Critical
//
// See: https://www.freedesktop.org/software/systemd/man/sd-daemon.html
var (
	// DebugPrefix is the prefix for debug-level log messages (systemd priority 7)
	DebugPrefix string = "<7>[DEBUG]    "
	// InfoPrefix is the prefix for info-level log messages (systemd priority 6)
	InfoPrefix string = "<6>[INFO]     "
	// WarnPrefix is the prefix for warning-level log messages (systemd priority 4)
	WarnPrefix string = "<4>[WARNING]  "
	// ErrPrefix is the prefix for error-level log messages (systemd priority 3)
	ErrPrefix string = "<3>[ERROR]    "
	// CritPrefix is the prefix for critical-level log messages (systemd priority 2)
	CritPrefix string = "<2>[CRITICAL] "
)

// Package-level logger instances for each log level.
// These are configured by Init() with appropriate flags and output destinations.
// Debug logs show no file info, Info/Warn show short file:line, Error/Crit show full path.
var (
	// DebugLog is the logger instance for debug-level messages
	DebugLog *log.Logger = log.New(DebugWriter, DebugPrefix, log.LstdFlags)
	// InfoLog is the logger instance for info-level messages
	InfoLog *log.Logger = log.New(InfoWriter, InfoPrefix, log.LstdFlags|log.Lshortfile)
	// WarnLog is the logger instance for warning-level messages
	WarnLog *log.Logger = log.New(WarnWriter, WarnPrefix, log.LstdFlags|log.Lshortfile)
	// ErrLog is the logger instance for error-level messages
	ErrLog *log.Logger = log.New(ErrWriter, ErrPrefix, log.LstdFlags|log.Llongfile)
	// CritLog is the logger instance for critical-level messages
	CritLog *log.Logger = log.New(CritWriter, CritPrefix, log.LstdFlags|log.Llongfile)
)

// loglevel stores the current log level setting
var loglevel string = "info"

// Init initializes cclogger with the specified log level and timestamp configuration.
//
// The lvl parameter accepts the following values:
//   - "debug": Show all log messages
//   - "info": Show info, warn, err, and crit messages (suppress debug)
//   - "warn": Show warn, err, and crit messages (suppress info and debug)
//   - "err" or "fatal": Show err and crit messages (suppress warn, info, and debug)
//   - "crit": Show only crit messages (suppress all others)
//
// If an invalid level is provided, all levels will be enabled (debug mode) and a
// warning will be logged.
//
// The logdate parameter controls timestamp inclusion:
//   - false: No timestamps (recommended when running under systemd, as journald adds timestamps)
//   - true: Include date and time in log output
//
// Example:
//
//	cclogger.Init("info", false) // Show info and above, no timestamps
func Init(lvl string, logdate bool) {
	switch lvl {
	case "crit":
		ErrWriter = io.Discard
		fallthrough
	case "err", "fatal":
		WarnWriter = io.Discard
		fallthrough
	case "warn":
		InfoWriter = io.Discard
		fallthrough
	case "info":
		DebugWriter = io.Discard
	case "debug":
		// Nothing to do - all writers remain active
		break
	default:
		// Use Error instead of Printf for consistency
		fmt.Fprintf(os.Stderr, "<3>[ERROR] cclogger: Invalid loglevel %q, using 'debug' (all levels enabled)\n", lvl)
		lvl = "debug" // Normalize to debug for storage
	}

	if !logdate {
		DebugLog = log.New(DebugWriter, DebugPrefix, 0)
		InfoLog = log.New(InfoWriter, InfoPrefix, log.Lshortfile)
		WarnLog = log.New(WarnWriter, WarnPrefix, log.Lshortfile)
		ErrLog = log.New(ErrWriter, ErrPrefix, log.Llongfile)
		CritLog = log.New(CritWriter, CritPrefix, log.Llongfile)
	} else {
		DebugLog = log.New(DebugWriter, DebugPrefix, log.LstdFlags)
		InfoLog = log.New(InfoWriter, InfoPrefix, log.LstdFlags|log.Lshortfile)
		WarnLog = log.New(WarnWriter, WarnPrefix, log.LstdFlags|log.Lshortfile)
		ErrLog = log.New(ErrWriter, ErrPrefix, log.LstdFlags|log.Llongfile)
		CritLog = log.New(CritWriter, CritPrefix, log.LstdFlags|log.Llongfile)
	}

	loglevel = lvl
}

// Loglevel returns the current loglevel
func Loglevel() string {
	return loglevel
}

// SetOutputFile redirects log output to a file for the specified level and all lower levels.
//
// The lvl parameter determines which loggers write to the file:
//   - "debug": Only debug logs go to file
//   - "info": Info and debug logs go to file
//   - "warn": Warn, info, and debug logs go to file
//   - "err" or "fatal": Error, warn, info, and debug logs go to file
//   - "crit": All logs go to file
//
// The file is opened in append mode and created if it doesn't exist.
// The file remains open for the lifetime of the loggers.
//
// WARNING: This function does not close the file after setting it as output.
// The file will remain open until the program exits. This is intentional to
// allow continued logging to the file.
//
// Example:
//
//	cclogger.SetOutputFile("warn", "/var/log/myapp.log")
//	// Now warn, info, and debug logs write to the file
//	// Error and critical logs still go to their default writers (stderr)
func SetOutputFile(lvl string, logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		log.Panic(err)
	}
	// Note: File is intentionally NOT closed - it must remain open for logging

	switch lvl {
	case "crit":
		CritLog.SetOutput(logFile)
		fallthrough
	case "err", "fatal":
		ErrLog.SetOutput(logFile)
		fallthrough
	case "warn":
		WarnLog.SetOutput(logFile)
		fallthrough
	case "info":
		InfoLog.SetOutput(logFile)
		fallthrough
	case "debug":
		DebugLog.SetOutput(logFile)
	default:
		// Use Error for consistency
		Error("cclogger.SetOutputFile: invalid loglevel", lvl)
	}
}

/* PRIVATE HELPER */

// Return unformatted string
func printStr(v ...any) string {
	return fmt.Sprint(v...)
}

// Return formatted string
func printfStr(format string, v ...any) string {
	return fmt.Sprintf(format, v...)
}

/* PRINT */

// Print logs to STDOUT without string formatting; application continues.
// Used for special cases not requiring log information like date or location.
func Print(v ...any) {
	fmt.Fprintln(os.Stdout, v...)
}

// Exit logs to STDOUT without string formatting; application exits with error code 0.
// Used for exiting succesfully with message after expected outcome, e.g. successful single-call application runs.
func Exit(v ...any) {
	fmt.Fprintln(os.Stdout, v...)
	os.Exit(0)
}

// Abort logs to STDOUT without string formatting; application exits with error code 1.
// Used for terminating with message after to be expected errors, e.g. wrong arguments or during init().
func Abort(v ...any) {
	fmt.Fprintln(os.Stdout, v...)
	os.Exit(1)
}

// ComponentPrint logs to INFO writer with a component prefix; application continues.
func ComponentPrint(component string, v ...any) {
	args := make([]any, 0, len(v)+1)
	args = append(args, fmt.Sprintf("[%s] ", component))
	args = append(args, v...)
	InfoLog.Output(2, printStr(args...))
}

// Debug logs to DEBUG writer without string formatting; application continues.
// Used for logging additional information, primarily for development.
func Debug(v ...any) {
	DebugLog.Output(2, printStr(v...))
}

// ComponentDebug logs to DEBUG writer with a component prefix; application continues.
func ComponentDebug(component string, v ...any) {
	args := make([]any, 0, len(v)+1)
	args = append(args, fmt.Sprintf("[%s] ", component))
	args = append(args, v...)
	DebugLog.Output(2, printStr(args...))
}

// Info logs to INFO writer without string formatting; application continues.
// Used for logging additional information, e.g. notable returns or common fail-cases.
func Info(v ...any) {
	InfoLog.Output(2, printStr(v...))
}

// ComponentInfo logs to INFO writer with a component prefix; application continues.
func ComponentInfo(component string, v ...any) {
	args := make([]any, 0, len(v)+1)
	args = append(args, fmt.Sprintf("[%s] ", component))
	args = append(args, v...)
	InfoLog.Output(2, printStr(args...))
}

// Warn logs to WARNING writer without string formatting; application continues.
// Used for logging important information, e.g. uncommon edge-cases or administration related information.
func Warn(v ...any) {
	WarnLog.Output(2, printStr(v...))
}

// ComponentWarn logs to WARNING writer with a component prefix; application continues.
func ComponentWarn(component string, v ...any) {
	args := make([]any, 0, len(v)+1)
	args = append(args, fmt.Sprintf("[%s] ", component))
	args = append(args, v...)
	WarnLog.Output(2, printStr(args...))
}

// Error logs to ERROR writer without string formatting; application continues.
// Used for logging errors, but code still can return default(s) or nil.
func Error(v ...any) {
	ErrLog.Output(2, printStr(v...))
}

// ComponentError logs to ERROR writer with a component prefix; application continues.
func ComponentError(component string, v ...any) {
	args := make([]any, 0, len(v)+1)
	args = append(args, fmt.Sprintf("[%s] ", component))
	args = append(args, v...)
	ErrLog.Output(2, printStr(args...))
}

// Fatal writes to CRITICAL writer without string formatting; application exits with error code 1.
// Used for terminating on unexpected errors with date and code location.
func Fatal(v ...any) {
	CritLog.Output(2, printStr(v...))
	os.Exit(1)
}

// Panic logs to PANIC function without string formatting; application exits with panic.
// Used for terminating on unexpected errors with stacktrace.
func Panic(v ...any) {
	panic(printStr(v...))
}

/* PRINT FORMAT*/

// Printf logs to STDOUT with string formatting; application continues.
// Used for special cases not requiring log information like date or location.
func Printf(format string, v ...any) {
	fmt.Fprintf(os.Stdout, format, v...)
}

// Exitf logs to STDOUT with string formatting; application exits with error code 0.
// Used for exiting succesfully with message after expected outcome, e.g. successful single-call application runs.
func Exitf(format string, v ...any) {
	fmt.Fprintf(os.Stdout, format, v...)
	os.Exit(0)
}

// Abortf logs to STDOUT with string formatting; application exits with error code 1.
// Used for terminating with message after to be expected errors, e.g. wrong arguments or during init().
func Abortf(format string, v ...any) {
	fmt.Fprintf(os.Stdout, format, v...)
	os.Exit(1)
}

// Debugf logs to DEBUG writer with string formatting; application continues.
// Used for logging additional information, primarily for development.
func Debugf(format string, v ...any) {
	DebugLog.Output(2, printfStr(format, v...))
}

// Infof log to INFO writer with string formatting; application continues.
// Used for logging additional information, e.g. notable returns or common fail-cases.
func Infof(format string, v ...any) {
	InfoLog.Output(2, printfStr(format, v...))
}

// Warnf logs to WARNING writer with string formatting; application continues.
// Used for logging important information, e.g. uncommon edge-cases or administration related information.
func Warnf(format string, v ...any) {
	WarnLog.Output(2, printfStr(format, v...))
}

// Errorf logs to ERROR writer with string formatting; application continues.
// Used for logging errors, but code still can return default(s) or nil.
func Errorf(format string, v ...any) {
	ErrLog.Output(2, printfStr(format, v...))
}

// Fatalf logs to CRITICAL writer with string formatting; application exits with error code 1.
// Used for terminating on unexpected errors with date and code location.
func Fatalf(format string, v ...any) {
	CritLog.Output(2, printfStr(format, v...))
	os.Exit(1)
}

// Panicf logs to PANIC function with string formatting; application exits with panic.
// Used for terminating on unexpected errors with stacktrace.
func Panicf(format string, v ...any) {
	panic(printfStr(format, v...))
}
