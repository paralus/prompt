package kube

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/RafaySystems/rcloud-base/pkg/audit"
	logv2 "github.com/RafaySystems/rcloud-base/pkg/log"
	"github.com/RafaySystems/prompt/pkg/prompt"
	"github.com/creack/pty"
)

var _log = logv2.GetLogger()
var brokers = []string{"kafka-core:9092"}
var producer, _ = audit.NewProducer(brokers)

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
func NewIOExecutor(rw io.ReadWriter, rows, cols uint16, args []string, event *audit.Event, kubectlBin, auditFile string) prompt.Executor {
	return func(ctx context.Context, s string) {
		s = strings.Trim(s, " ")
		if s == "" {
			return
		}

		createKubectlCommandAudit(event, "kubectl "+s, auditFile)

		// handle prompt clear
		if strings.Index(s, "clear") >= 0 {
			// clear | hexdump
			rw.Write([]byte{0x1b, 0x5b, 0x48, 0x1b, 0x5b, 0x32, 0x4a})
			return
		}

		var execArgs []string
		for _, arg := range args {
			if strings.TrimSpace(arg) != "" {
				execArgs = append(execArgs, arg)
			}
		}

		for _, arg := range strings.Split(s, " ") {
			if strings.TrimSpace(arg) != "" {
				execArgs = append(execArgs, arg)
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

			go func() {
				defer wg.Done()
				_, err := io.Copy(rw, f)
				_log.Infow("exited copy from pty", "error", err)
			}()
			go func() {
				defer wg.Done()
				_, err := io.Copy(f, rw)
				_log.Infow("exited copy to pty", "error", err)
			}()

			cmd.Wait()
			f.Close()
			wg.Wait()
			return
		}

		_log.Debugw("executing non interative kubectl", "args", execArgs)

		if len(execArgs) > 3 && execArgs[2] == "config" {
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
func createKubectlCommandAudit(event *audit.Event, command string, auditFile string) {
	if event == nil {
		_log.Errorw("Event is nil")
		return
	}
	event.Detail.Message = command
	event.Version = audit.VersionV1
	event.Category = audit.AuditCategory
	event.Origin = audit.OriginCluster

	// Dump event to audit.log file
	data, err := json.Marshal(event)
	if err != nil {
		_log.Errorw("unable to format event to JSON", "error", err)
		return
	}

	f, err := os.OpenFile(auditFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	defer f.Close()
	if err != nil {
		_log.Errorw("unable to open audit file", "error", err)
		return
	}
	_, err = f.Write(append(data, '\n'))
	if err != nil {
		_log.Errorw("unable to write to audit file", "error", err)
		return
	}
}
