package kube

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/mattn/go-shellwords"
	"github.com/paralus/paralus/pkg/audit"
	logv2 "github.com/paralus/paralus/pkg/log"
	"github.com/paralus/prompt/pkg/prompt"
	"go.uber.org/zap"
)

var _log = logv2.GetLogger()

func isInteractive(s string) bool {
	switch {
	case strings.Index(s, "exec") >= 0:
	case strings.Index(s, "logs") >= 0:
	case strings.Index(s, "edit") >= 0:
	case strings.Index(s, "-w") >= 0:
	default:
		return false
	}

	return true
}

// NewIOExecutor returns executor tied to io ReadWriter
func NewIOExecutor(rw io.ReadWriter, rows, cols uint16, args []string, event *audit.Event, kubectlBin string, auditLogger *zap.Logger) prompt.Executor {
	return func(ctx context.Context, s string) {
		s = strings.Trim(s, " ")
		if s == "" {
			return
		}

		createKubectlCommandAudit(event, "kubectl "+s, auditLogger)

		// handle prompt clear
		if strings.Index(s, "clear") >= 0 {
			// clear | hexdump
			rw.Write([]byte{0x1b, 0x5b, 0x48, 0x1b, 0x5b, 0x32, 0x4a})
			return
		}

		var execArgs []string

		// Add kubectl flags first
		for _, arg := range args {
			if strings.TrimSpace(arg) != "" {
				execArgs = append(execArgs, arg)
			}
		}

		// appending kubectl commands to execute
		p, err := shellwords.Parse(s)
		if err != nil {
			_log.Error("unable to parse command", zap.Error(err))
			return
		}

		// Add all parsed arguments first
		execArgs = append(execArgs, p...)

		// Handle namespace flag specially - look for it in any position
		for i := 0; i < len(execArgs); i++ {
			if execArgs[i] == "-n" && i+1 < len(execArgs) {
				// Move namespace flag and its value to the beginning of the command
				// after the initial kubectl flags
				nsFlag := execArgs[i]
				nsValue := execArgs[i+1]
				// Remove the flag and value from their current position
				execArgs = append(execArgs[:i], execArgs[i+2:]...)
				// Insert them after the initial kubectl flags
				initialFlagsLen := len(args)
				execArgs = append(execArgs[:initialFlagsLen], append([]string{nsFlag, nsValue}, execArgs[initialFlagsLen:]...)...)
				break // Only handle the first occurrence of -n
			}
		}

		if isInteractive(s) {
			_log.Debugw("executing interactive kubectl", "args", s)

			cmd := exec.CommandContext(ctx, kubectlBin, execArgs...)
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, "KUBE_EDITOR=vim")

			f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
			if err != nil {
				rw.Write([]byte(err.Error()))
				rw.Write([]byte{'\r', '\n'})
				return
			}

			var wg sync.WaitGroup
			wg.Add(2)

			// Create a done channel to signal goroutines to stop
			done := make(chan struct{})

			// Handle cleanup on exit
			defer func() {
				close(done)
				wg.Wait()
				f.Close()
				// Send a newline to ensure prompt is on a new line
				rw.Write([]byte{'\r', '\n'})
			}()

			// Copy from PTY to websocket
			go func() {
				defer wg.Done()
				buf := make([]byte, 1024)
				for {
					select {
					case <-done:
						return
					default:
						n, err := f.Read(buf)
						if err != nil {
							if err != io.EOF {
								_log.Infow("error reading from pty", "error", err)
							}
							return
						}
						if n > 0 {
							if _, err := rw.Write(buf[:n]); err != nil {
								_log.Infow("error writing to websocket", "error", err)
								return
							}
						}
					}
				}
			}()

			// Copy from websocket to PTY
			go func() {
				defer wg.Done()
				buf := make([]byte, 1024)
				for {
					select {
					case <-done:
						return
					default:
						n, err := rw.Read(buf)
						if err != nil {
							if err != io.EOF {
								_log.Infow("error reading from websocket", "error", err)
							}
							return
						}
						if n > 0 {
							if _, err := f.Write(buf[:n]); err != nil {
								_log.Infow("error writing to pty", "error", err)
								return
							}
						}
					}
				}
			}()

			// Wait for command to complete
			err = cmd.Wait()
			if err != nil {
				_log.Infow("command exited with error", "error", err)
			}

			return
		}

		_log.Debugw("executing non interative kubectl", "args", execArgs)

		if len(execArgs) > 3 && execArgs[0] == "config" {
			// filter raw/flattern argument to avoid displaying cert data
			for _, s := range execArgs {
				if s == "--raw" || s == "--flatten" {
					return
				}
			}
		}

		cmd := exec.CommandContext(ctx, kubectlBin, execArgs...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			_log.Infow("unable to run command", "error", err)
		}
		_log.Infow("executed non interative kubectl", "args", execArgs)
		out = bytes.ReplaceAll(out, []byte{'\n'}, []byte{'\r', '\n'})
		_, err = rw.Write(out)
		if err != nil {
			_log.Infow("unable to write output", "error", err)
		}

		return
	}
}

// createKubectlCommandAudit send the kubectl command audit event to the audit.log file
func createKubectlCommandAudit(event *audit.Event, command string, auditLogger *zap.Logger) {
	if event == nil {
		_log.Errorw("Event is nil")
		return
	}
	event.Detail.Message = command
	event.Version = audit.VersionV1
	event.Category = audit.AuditCategory
	event.Origin = audit.OriginCluster

	go audit.WriteEvent(event, auditLogger)
}
