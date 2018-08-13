/* Copyright (c) 2017 Gregor Riepl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package util

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

const (
	// signalQueueLength specifies the maximum number of unhandled control signals
	signalQueueLength int = 100
	// logQueueLength specifies the maximum number of unwritten log messages
	logQueueLength int = 100
	// timeFormat configures the format for time strings
	timeFormat string = time.RFC3339
	// hupSignal is a signal identifier for a "reopen the log" notification.
	// Distinct from UserSignal.
	hupSignal internalSignal = internalSignal("HUP")
	// shutdownSignal is a signal identifier for a "stop logging" notification.
	shutdownSignal internalSignal = internalSignal("SDN")
	// srcFileUnknown is the string stored in KeySrcFile when the caller's file name cannot be determined
	srcFileUnknown string = "<UNKNOWN>"
	// srcLineUnknown is the number stored in KeySrcLine when the caller's source code line number cannot be determined
	srcLineUnknown int = 0
	//
	// KeyModule is the standard key for a user-defined module name
	KeyModule string = "module"
	// KeyTime is the standard key for the time stamp when the log entry was generated
	KeyTime string = "time"
)

var (
	globalStandardLogger MultiLogger = MultiLogger{
		&ConsoleLogger{},
	}
)

type internalSignal string

func (s internalSignal) Signal() {}
func (s internalSignal) String() string {
	return string(s)
}

// Dict is a generic string:any dictionary type, for more convenience
// when creating structured logs.
type Dict map[string]interface{}

// Logger is an interface for loggers that can generate JSON-formatted logs
// from structured data.
//
// It is recommended that logs follow some general guidelines, like adding
// a reference to the module that generated them, or a flag to differentiate
// various kinds of log messages.
//
// See ModuleLogger for an easy way to do this.
//
// Examples:
// { "module": "client", "type": "connect", "stream": "http://test.url/" }
// { "module": "connection", "type": "connect", "source": "1.2.3.4:49999", "url": "/stream" }
// { "module": "connection", "type": "disconnect", "source": "1.2.3.4:49999", "url": "/stream", "duration": 61, "bytes": 12087832 }
type Logger interface {
	// Logd writes one or multiple data structures to the log represented by this logger.
	// Each argument is processed through json.Marshal and generates one line in the log.
	//
	// Example usage:
	//   logger.Logd(Dict{ "key": "value" }, Dict{ "key": "value2" })
	Logd(lines ...Dict)
	// Log is a convenience function that sends a single log line to the logger.
	// The arguments are alternating key -> value pairs that are assembled into a dictionary.
	//
	// This function is slightliy easier to use in many cases, because it doesn't require
	// in-place creation of data structures. But it's usually also slower.
	// Simply call:
	//   logger.Log("key", "value", "key2", 10)
	Logkv(keyValues ...interface{})
}

// LogFunnel is a simple helper for converting variadic key-value pairs into a dictionary
func LogFunnel(keyValues []interface{}) Dict {
	d := make(Dict)
	// we need an even number of additional args
	for i := 0; i+1 < len(keyValues); i += 2 {
		k, ok := keyValues[i].(string)
		// ignore if the key is not a string
		if ok {
			d[k] = keyValues[i+1]
		}
	}
	return d
}

// NewGlobalModuleLogger creates a global logger for the current package and
// connects it to the global standard logger.
//
// The default output for standard logger is a JSON log with added timestamps,
// but this can be changed by calling SetGlobalStandardLogger.
//
// An optional dictionary argument allows specifying additional keys that are
// added to every log line. Can be nil if you don't need it.
func NewGlobalModuleLogger(module string, dict Dict) Logger {
	more := make(Dict)
	for k, v := range dict {
		more[k] = v
	}
	more[KeyModule] = module
	logger := &ModuleLogger{
		Logger:   globalStandardLogger,
		Defaults: more,
	}
	return logger
}

// SetGlobalStandardLogger assigns a new backing logger to the global standard logger
//
// A reference to the old logger is returned.
func SetGlobalStandardLogger(logger Logger) Logger {
	old := globalStandardLogger[0]
	globalStandardLogger[0] = logger
	return old
}

// ModuleLogger encapsulates default values for a JSON log.
//
// This simplifies log calls, as values like the current module can be
// initialised once, and they will be reused on every log call.
//
// If AddTimestamp is true, each log line will contain the key 'time' with the
// current time in RFC 3339 format (ex.: 2006-01-02T15:04:05Z07:00).
//
// The keys in the Defaults dictionary will always be added.
//
// It is highly recommended to add at least a 'module' key to the defaults,
// so the logging module can be identified.
type ModuleLogger struct {
	// Logger is the backing logger to send log lines to.
	Logger Logger
	// Defaults is a dictionary containing default keys.
	// It is highly recommended to add any immutable data here,
	// in particular the key 'module' with a unique name for the module sending the log
	// will be very useful.
	Defaults Dict
	// AddTimestamp determines if a "time" value with the current time in RFC 3339 format
	// is added to the dictionary before it is passed to the underlying logger.
	AddTimestamp bool
}

// Log adds predefined values to each log line and writes it to the encapsulated log.
func (logger *ModuleLogger) Logd(lines ...Dict) {
	proclines := make([]Dict, len(lines))
	for i, line := range lines {
		processed := make(Dict)
		for key, value := range logger.Defaults {
			processed[key] = value
		}
		if logger.AddTimestamp {
			processed[KeyTime] = time.Now().Format(timeFormat)
		}
		for key, value := range line {
			processed[key] = value
		}
		proclines[i] = processed
	}
	logger.Logger.Logd(proclines...)
}

func (logger *ModuleLogger) Logkv(keyValues ...interface{}) {
	logger.Logd(LogFunnel(keyValues))
}

// DummyLogger is a logger placeholder that doesn't actually log anything.
// Just a placeholder for the real big boy loggers.
type DummyLogger struct{}

func (*DummyLogger) Logd(lines ...Dict)             {}
func (*DummyLogger) Logkv(keyValues ...interface{}) {}

// Multilogger logs to several backend loggers at once.
type MultiLogger []Logger

// Log writes the same log lines to all backing loggers.
func (logger MultiLogger) Logd(lines ...Dict) {
	for _, backer := range logger {
		backer.Logd(lines...)
	}
}

func (logger MultiLogger) Logkv(keyValues ...interface{}) {
	logger.Logd(LogFunnel(keyValues))
}

// ConsoleLogger is a simple logger that prints to stdout.
type ConsoleLogger struct{}

// Log writes a log line to stdout.
//
// Your best bet if you don't want/need a full-blown file logging queue with
// signal-initiated reopening or a central logging server.
func (*ConsoleLogger) Logd(lines ...Dict) {
	encoder := json.NewEncoder(os.Stdout)
	for _, line := range lines {
		err := encoder.Encode(line)
		if err != nil {
			fmt.Printf("{\"event\":\"error\",\"message\":\"Cannot encode log line\",\"line\":\"%v\"}\n", line)
		}
	}
}

func (logger *ConsoleLogger) Logkv(keyValues ...interface{}) {
	logger.Logd(LogFunnel(keyValues))
}

// A FileLogger writes JSON-formatted log lines to a file.
//
// Log lines are prefixed with a timestamp in RFC3339 format, like this:
// [2006-01-02T15:04:05Z07:00] <JSON>
type FileLogger struct {
	// notification channel
	// also used for system signals
	signals chan os.Signal
	// log file name
	name string
	// log file handle
	log io.WriteCloser
	// message queue
	messages chan interface{}
	// log line counter
	lines uint64
	// dropped line counter
	drops uint64
	// error counter (encoding errors or closed log file)
	errors uint64
}

// NewFileLogger creates a new FileLogger and optionally installs a SIGUSR1 handler;
// pass sigusr=true for that purpose. This is useful for log rotation, etc.
//
// Signals are only fully supported on POSIX systems, so no SIGUSR1 is sent
// when running on Microsoft Windows, for example. The signal handler is
// still installed, but it is never notified.
func NewFileLogger(logfile string, sigusr bool) (*FileLogger, error) {
	// create logger instance
	logger := &FileLogger{
		signals:  make(chan os.Signal, signalQueueLength),
		name:     logfile,
		messages: make(chan interface{}, logQueueLength),
	}

	// open the log for the first time
	err := logger.reopenLog()
	if err != nil {
		return nil, err
	}

	// install signal handler and start listening thread
	RegisterUserSignalHandler(logger.signals)
	go logger.handle()

	return logger, nil
}

// Log writes a series of log lines, prefixed by a time stamp in RFC3339 format.
func (logger *FileLogger) Logd(lines ...Dict) {
	// send these down the queue
	for _, line := range lines {
		select {
		case logger.messages <- line:
			// ok
		default:
			fmt.Printf("{\"event\":\"error\",\"message\":\"Log queue is full, message dropped\",\"line\":\"%v\"}\n", line)
			logger.drops++
		}
	}
}

func (logger *FileLogger) Logkv(keyValues ...interface{}) {
	logger.Logd(LogFunnel(keyValues))
}

// Writes a single log line
func (logger *FileLogger) writeLog(line interface{}) {
	// only log if the output is open
	if logger.log != nil {
		data, err := json.Marshal(line)
		if err == nil {
			format := fmt.Sprintf("[%s] %s\n", time.Now().Format(timeFormat), data)
			logger.log.Write([]byte(format))
			logger.lines++
		} else {
			fmt.Printf("{\"event\":\"error\",\"message\":\"Cannot encode log line\",\"line\":\"%v\"}\n", line)
			logger.errors++
		}
	} else {
		fmt.Printf("{\"event\":\"error\",\"message\":\"Output is closed, dropping line\",\"line\":\"%v\"}\n", line)
		logger.errors++
	}
}

// Closes the log file and disables further logging.
func (logger *FileLogger) Close() {
	fmt.Printf("{\"event\":\"close_signal\",\"message\":\"Closing log\"}\n")
	logger.signals <- hupSignal
}

// Closes the log and stops/removes the signal handler
func (logger *FileLogger) closeLog() error {
	fmt.Printf("{\"event\":\"close\",\"message\":\"Really closing log\"}\n")

	// uninstall the singal handler
	signal.Stop(logger.signals)
	// signal stop
	logger.signals <- shutdownSignal

	// close the log
	err := logger.log.Close()
	logger.log = nil

	return err
}

// (Re-)opens the log file.
func (logger *FileLogger) reopenLog() error {
	fmt.Printf("{\"event\":\"reopen\",\"message\":\"Reopening log\"}\n")

	var err error = nil

	if logger.log != nil {
		// close first
		err = logger.log.Close()
		logger.log = nil
	}
	if err == nil {
		logger.log, err = os.OpenFile(logger.name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0666))
	}

	return err
}

// Handles the log queue and the USR1 signal.
// If USR1 is received the log file is closed and reopened.
func (logger *FileLogger) handle() {
	running := true

	for running {
		select {
		case signal := <-logger.signals:
			// check signal type
			switch signal {
			case UserSignal:
				// reopen the log file
				err := logger.reopenLog()
				if err != nil {
					// if this fails, print a message to the standard log
					fmt.Printf("{\"event\":\"error\",\"message\":\"Error reopening log\",\"error\":\"reopen\",\"errmsg\":\"%s\"}\n", err.Error())
				}
			case hupSignal:
				// reopen the log file
				err := logger.closeLog()
				if err != nil {
					// if this fails, print a message to the standard log
					fmt.Printf("{\"event\":\"error\",\"message\":\"Error reopening log\",\"error\":\"reopen\",\"errmsg\":\"%s\"}\n", err.Error())
				}
			case shutdownSignal:
				// shutdown requested
				running = false
				fmt.Printf("{\"event\":\"shutdown\",\"message\":\"Shutting down logger\"}\n")
			}
		case line := <-logger.messages:
			// encode and write the next line
			logger.writeLog(line)
		}
	}
}
