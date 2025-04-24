// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package cclogger implements a simple log wrapper for the standard log package
//
// cclogger provides a simple way of logging with different levels.
// Time/Date are not logged because systemd adds
// them (default, can be changed by setting logdate to true.
// Additionally log output can be set to a file. Default output is stderr.
// Uses these prefixes: https://www.freedesktop.org/software/systemd/man/sd-daemon.html
package cclogger

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	DebugWriter io.Writer = os.Stderr
	InfoWriter  io.Writer = os.Stderr
	WarnWriter  io.Writer = os.Stderr
	ErrWriter   io.Writer = os.Stderr
	CritWriter  io.Writer = os.Stderr
)

var (
	DebugPrefix string = "<7>[DEBUG]    "
	InfoPrefix  string = "<6>[INFO]     "
	WarnPrefix  string = "<4>[WARNING]  "
	ErrPrefix   string = "<3>[ERROR]    "
	CritPrefix  string = "<2>[CRITICAL] "
)

var (
	DebugLog *log.Logger = log.New(DebugWriter, DebugPrefix, log.LstdFlags)
	InfoLog  *log.Logger = log.New(InfoWriter, InfoPrefix, log.LstdFlags|log.Lshortfile)
	WarnLog  *log.Logger = log.New(WarnWriter, WarnPrefix, log.LstdFlags|log.Lshortfile)
	ErrLog   *log.Logger = log.New(ErrWriter, ErrPrefix, log.LstdFlags|log.Llongfile)
	CritLog  *log.Logger = log.New(CritWriter, CritPrefix, log.LstdFlags|log.Llongfile)
)

var loglevel string = "info"

// Init initializes cclogger. lvl indicates the loglevel with such values as
// "debug", "info", "warn", "err", "fatal", "crit". If logdate is set to true a
// date and time is added to the log output.
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
		// Nothing to do...
		break
	default:
		fmt.Printf("pkg/log: Flag 'loglevel' has invalid value %#v\npkg/log: Will use default loglevel 'debug'\n", lvl)
		// SetLogLevel("debug")
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

// SetOutputFile sets the output of selected loglevels to a file indicated by
// the logfile function argument. All loggers lower than lvl are set to the
// output file. Example: If lvl is warn, the warn, info, and debug loggers
// will write to logfile.
func SetOutputFile(lvl string, logfile string) {
	logFile, err := os.OpenFile(logfile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()

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
		fmt.Printf("pkg/log: Flag 'loglevel' has invalid value %#v\npkg/log\n", lvl)
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

func ComponentPrint(component string, v ...any) {
	InfoLog.Print(fmt.Sprintf("[%s] ", component), v)
}

// Debug logs to DEBUG writer without string formatting; application continues.
// Used for logging additional information, primarily for development.
func Debug(v ...any) {
	DebugLog.Output(3, printStr(v...))
}

func ComponentDebug(component string, v ...any) {
	DebugLog.Print(fmt.Sprintf("[%s] ", component), v)
}

// Info logs to INFO writer without string formatting; application continues.
// Used for logging additional information, e.g. notable returns or common fail-cases.
func Info(v ...any) {
	InfoLog.Output(3, printStr(v...))
}

func ComponentInfo(component string, v ...any) {
	InfoLog.Print(fmt.Sprintf("[%s] ", component), v)
}

// Warn logs to WARNING writer without string formatting; application continues.
// Used for logging important information, e.g. uncommon edge-cases or administration related information.
func Warn(v ...any) {
	WarnLog.Output(3, printStr(v...))
}

func ComponentWarn(component string, v ...any) {
	WarnLog.Print(fmt.Sprintf("[%s] ", component), v)
}

// Error logs to ERROR writer without string formatting; application continues.
// Used for logging errors, but code still can return default(s) or nil.
func Error(v ...any) {
	ErrLog.Output(3, printStr(v...))
}

func ComponentError(component string, v ...any) {
	ErrLog.Print(fmt.Sprintf("[%s] ", component), v)
}

// Fatal writes to CRITICAL writer without string formatting; application exits with error code 1.
// Used for terminating on unexpected errors with date and code location.
func Fatal(v ...any) {
	CritLog.Output(3, printStr(v...))
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
	DebugLog.Output(3, printfStr(format, v...))
}

// Infof log to INFO writer with string formatting; application continues.
// Used for logging additional information, e.g. notable returns or common fail-cases.
func Infof(format string, v ...any) {
	InfoLog.Output(3, printfStr(format, v...))
}

// Warnf logs to WARNING writer with string formatting; application continues.
// Used for logging important information, e.g. uncommon edge-cases or administration related information.
func Warnf(format string, v ...any) {
	WarnLog.Output(3, printfStr(format, v...))
}

// Errorf logs to ERROR writer with string formatting; application continues.
// Used for logging errors, but code still can return default(s) or nil.
func Errorf(format string, v ...any) {
	ErrLog.Output(3, printfStr(format, v...))
}

// Fatalf logs to CRITICAL writer with string formatting; application exits with error code 1.
// Used for terminating on unexpected errors with date and code location.
func Fatalf(format string, v ...any) {
	CritLog.Output(3, printfStr(format, v...))
	os.Exit(1)
}

// Panicf logs to PANIC function with string formatting; application exits with panic.
// Used for terminating on unexpected errors with stacktrace.
func Panicf(format string, v ...any) {
	panic(printfStr(format, v...))
}
