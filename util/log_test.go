/* Copyright (c) 2018 Gregor Riepl
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
	"bufio"
	"bytes"
	"os"
	"sync"
	"testing"
)

func TestInternalSignal00(t *testing.T) {
	t00 := internalSignal("t00")
	// nothing should happen here
	t00.Signal()
	if t00.String() != "t00" {
		t.Errorf("Signal does not convert to identity")
	}
}

func TestLogFunnel00(t *testing.T) {
	t00 := []interface{}{
		"a", "b",
		"bb", 10,
		100, "x",
		"cde",
	}
	r00 := LogFunnel(t00)
	if v, ok := r00["a"]; !ok || v != "b" {
		t.Errorf("Key a not represented correctly in dictionary")
	}
	if v, ok := r00["bb"]; !ok || v != 10 {
		t.Errorf("Key bb not represented correctly in dictionary")
	}
	// 100 cannot be in dict - type mismatch
	if _, ok := r00["100"]; ok {
		t.Errorf("Key 100 shouldn't be in dictionary")
	}
	if _, ok := r00["cde"]; ok {
		t.Errorf("Key cde shouldn't be in dictionary")
	}
}

type mockLogger struct {
	t     *testing.T
	lines []Dict
}

func (l *mockLogger) Logd(lines ...Dict) {
	l.lines = append(l.lines, lines...)
}
func (l *mockLogger) Logkv(keyValues ...interface{}) {
	l.Logd(LogFunnel(keyValues))
}

func TestGlobalStdLogger00(t *testing.T) {
	m00a := &mockLogger{
		t: t,
	}
	SetGlobalStandardLogger(m00a)
	m00b := &mockLogger{
		t: t,
	}
	r00 := SetGlobalStandardLogger(m00b)
	if r00 != m00a {
		t.Errorf("Original logger is not the same as the returned logger")
	}
}

func TestGlobalModuleLogger00(t *testing.T) {
	m00 := &mockLogger{
		t: t,
	}
	SetGlobalStandardLogger(m00)
	logger := NewGlobalModuleLogger("t00", Dict{
		"cde": "v00",
	})
	logger.Logkv("aa", "bb")
	if len(m00.lines) != 1 {
		t.Fatalf("Couldn't find the correct number of log lines in output")
	}
	if m00.lines[0]["aa"] != "bb" {
		t.Errorf("Didn't find test key in log line")
	}
	if m00.lines[0][KeyModule] != "t00" {
		t.Errorf("Didn't find module key in log line")
	}
	if m00.lines[0]["cde"] != "v00" {
		t.Errorf("Didn't find custom module key in log line")
	}
}

func TestModuleLogger00(t *testing.T) {
	m00 := &mockLogger{
		t: t,
	}
	l00 := &ModuleLogger{
		Logger:       m00,
		AddTimestamp: true,
	}
	l00.Logkv("a", "b")
	if len(m00.lines) != 1 {
		t.Fatalf("Couldn't find the correct number of log lines in output")
	}
	if m00.lines[0]["a"] != "b" {
		t.Errorf("Didn't find test key in log line")
	}
	if len(m00.lines[0][KeyTime].(string)) < 20 {
		t.Errorf("Missing time key, or formatted time too short")
	}
}

func TestMultiLogger00(t *testing.T) {
	m00a := &mockLogger{
		t: t,
	}
	m00b := &mockLogger{
		t: t,
	}
	l00 := MultiLogger{
		m00a,
		m00b,
	}
	l00.Logkv("ml00", "vl00")
	if len(m00a.lines) != 1 || len(m00b.lines) != 1 {
		t.Fatalf("Couldn't find the correct number of log lines in output")
	}
	if m00a.lines[0]["ml00"] != "vl00" {
		t.Errorf("Didn't find test key in log line of first logger")
	}
	if m00b.lines[0]["ml00"] != "vl00" {
		t.Errorf("Didn't find test key in log line of second logger")
	}
}

func TestConsoleLogger00(t *testing.T) {
	m00 := &mockLogger{
		t: t,
	}
	r00, w00, err := os.Pipe()
	if err != nil {
		t.Fatalf("Cannot create pipe: %v", err)
	}
	wg := &sync.WaitGroup{}
	go func(l Logger, r *os.File, wg *sync.WaitGroup) {
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadBytes('\n')
			if err != nil {
				break
			}
			l.Logkv("line", line)
			wg.Done()
		}
	}(m00, r00, wg)
	o00 := os.Stdout
	defer func(r, w, o *os.File) {
		defer r.Close()
		defer w.Close()
		os.Stdout = o
	}(r00, w00, o00)
	os.Stdout = w00
	l00 := &ConsoleLogger{}
	wg.Add(1)
	l00.Logkv("key00", "value00")
	wg.Wait()
	if len(m00.lines) != 1 {
		t.Fatalf("Couldn't find the correct number of log lines in output")
	}
	if !bytes.Contains(m00.lines[0]["line"].([]byte), []byte("key00")) {
		t.Errorf("Didn't find test key in log line: %s", m00.lines[0])
	}
	if !bytes.Contains(m00.lines[0]["line"].([]byte), []byte("value00")) {
		t.Errorf("Didn't find test value in log line: %s", m00.lines[0])
	}
}
