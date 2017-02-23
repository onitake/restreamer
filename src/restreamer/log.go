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

package restreamer

import (
	"os"
	"log"
	"sync"
	"syscall"
	"os/signal"
	"encoding/json"
)

// JsonLogger is an interface for loggers that can generate JSON-formatted logs.
type JsonLogger interface {
	// Writes one or multiple date structures to the log represented by this logger.
	// Each argument is processed through json.Marshal and generates one line in the log.
	Log(json ...interface{}) error
}

// A FileLogger allows writing JSON-formatted log lines to a file.
type FileLogger struct {
	// OS signal notification channel
	// also used for internal shutdown
	signals chan os.Signal
	// synchronization mutex (for reopening and writing)
	lock sync.Mutex
	// log file name
	name string
	// log file handle
	log *os.File
	// JSON writer
	encoder *json.Encoder
	// log line counter
	lines uint64
	// dropped line counter
	drops uint64
}

// NewLogger creates a new FileLogger and optionally installs a SIGUSR1 signal handler;
// pass sigusr=true for that purpose. This is useful for log rotation, etc.
//
// Signals are only fully supported on POSIX systems, so no SIGUSR1 is sent
// when running on Microsoft Windows, for example. The signal handler is
// still installed, but it is never notified.
func NewFileLogger(logfile string, sigusr bool) (*FileLogger, error) {
	// create logger instance
	logger := &FileLogger{
		signals: make(chan os.Signal, 1),
		name: logfile,
	}
	
	// install signal handler and start listening thread
	logger.signals = make(chan os.Signal, 1)
	signal.Notify(logger.signals, syscall.SIGUSR1)
	go logger.handle()
	
	// open the log for the first time
	err := logger.Reopen()
	
	return logger, err
}

func (logger *FileLogger) Log(json ...interface{}) {
	// lock first
	logger.lock.Lock()
	
	// only log if the output is open
	if logger.encoder != nil {
		for line := range json {
			logger.encoder.Encode(line)
			logger.lines++
		}
	} else {
		lines := len(json)
		log.Printf("%d JSON log lines dropped!", lines)
		logger.drops += uint64(lines)
	}
	
	// and done
	logger.lock.Unlock()
}

// Closes the log and stops/removes the signal handler
func (logger *FileLogger) Close() error {
	// lock first
	logger.lock.Lock()
	
	// uninstall the singal handler
	signal.Stop(logger.signals)
	// signal stop
	logger.signals <-os.Interrupt
	
	// close the log
	err := logger.log.Close()
	logger.encoder = nil
	logger.log = nil
	
	// and done
	logger.lock.Unlock()
	
	return err
}

// (Re-)opens the log file.
func (logger *FileLogger) Reopen() error {
	var err error = nil
	
	// lock first
	logger.lock.Lock()
	
	if logger.log != nil {
		// close first
		err = logger.log.Close()
		logger.log = nil
		logger.encoder = nil
	}
	if err != nil {
		logger.log, err = os.OpenFile(logger.name, os.O_WRONLY | os.O_APPEND | os.O_CREATE, os.FileMode(0666))
	}
	if err != nil {
		logger.encoder = json.NewEncoder(logger.log)
	}
	
	// and done
	logger.lock.Unlock()
	
	return err
}

// Receives and handles the USR1 signal, prompting closure and
// reopening of the log file.
func (logger *FileLogger) handle() {
	running := true
	
	for running {
		select {
			case signal := <-logger.signals:
				// check signal type
				switch signal {
					case syscall.SIGUSR1:
						// reopen the log file
						err := logger.Reopen()
						// if this fails, print a message to the standard log
						log.Printf("Error reopening log: %s", err)
					case os.Interrupt:
						// shutdown requested
						running = false
				}
		}
	}
}
