package protocol

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

type ForkReader struct {
	command    *exec.Cmd
	dataInput  io.ReadCloser
	errorInput io.ReadCloser
}

func (f *ForkReader) Read(p []byte) (n int, err error) {
	return f.dataInput.Read(p)
}

func (f *ForkReader) Close() error {
	return f.command.Process.Kill()
}

func NewForkReader(command string, arguments []string) (*ForkReader, error) {
	cmd := exec.Command(command, arguments...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	logger.Logkv(
		"event", eventForkStarted,
		"pid", cmd.Process.Pid,
		"command", cmd.Path,
		"message", fmt.Sprintf("Fork reader command started: %s %v", command, arguments),
	)
	fr := &ForkReader{
		command:    cmd,
		dataInput:  stdout,
		errorInput: stderr,
	}
	// Launch a goroutine that logs output to stderr
	go func(f *ForkReader) {
		buffer := bufio.NewReader(f.errorInput)
		for line, err := "", error(nil); err == nil; {
			line, err = buffer.ReadString('\n')
			if err != nil && err != io.EOF {
				logger.Logkv(
					"event", eventForkError,
					"error", errorForkStderrRead,
					"command", f.command.Path,
					"message", fmt.Sprintf("Error reading from stderr: %v", err),
				)
			}
			logger.Logkv(
				"event", eventForkChildMessage,
				"command", f.command.Path,
				"message", line,
			)
		}
	}(fr)
	// Wait for command exit in a goroutine, so we can report process exit asynchronously
	go func(f *ForkReader) {
		err := f.command.Wait()
		if err != nil {
			logger.Logkv(
				"event", eventForkError,
				"error", errorForkExit,
				"exitcode", cmd.ProcessState.ExitCode(),
				"command", f.command.Path,
				"message", fmt.Sprintf("Process exited with error: %v", err),
			)
		}
	}(fr)
	return fr, nil
}
