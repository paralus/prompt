package kube

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/RafaySystems/rafay-common/pkg/audit"
	logv2 "github.com/RafaySystems/rafay-common/pkg/log/v2"
	"github.com/RafaySystems/rafay-prompt/pkg/prompt"
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
func NewIOExecutor(rw io.ReadWriter, rows, cols uint16, args []string, event *audit.Event) prompt.Executor {
	return func(ctx context.Context, s string) {
		s = strings.Trim(s, " ")
		if s == "" {
			return
		}

		createKubectlCommandAudit(event, "kubectl "+s)

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

			cmd := exec.CommandContext(ctx, "/opt/rafay/kubectl", execArgs...)
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

		cmd := exec.CommandContext(ctx, "/opt/rafay/kubectl", execArgs...)
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

// createKubectlCommandAudit send the kubectl command audit event to the kafka
func createKubectlCommandAudit(event *audit.Event, command string) {
	if event == nil {
		_log.Errorw("Event is nil")
		return
	}
	if producer == nil {
		var err error
		producer, err = audit.NewProducer(brokers)
		if err != nil {
			_log.Errorw("unable to create kafka producer", "error", err)
			return
		}
	}
	event.Detail.Message = command
	err := audit.CreateEvent(event,
		audit.WithPartnerID(event.PartnerID),
		audit.WithOrganizationID(event.OrganizationID),
		audit.WithProjectID(event.ProjectID),
		audit.WithOrigin(audit.OriginCluster),
		audit.WithVersion(audit.VersionV1),
		audit.WithCategory(audit.AuditCategory),
		audit.WithProducer(producer),
		audit.WithTopic("relay-commands"),
	)
	if err != nil {
		_log.Errorw("unable to create audit event", "error", err)
		return
	}
}
