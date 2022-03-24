package ptyio

import (
	"context"
	"io"
	"os/exec"

	logv2 "github.com/RafayLabs/rcloud-base/pkg/log"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var (
	_log = logv2.GetLogger()
)

func New(ctx context.Context, conn *websocket.Conn, rows, cols uint16) {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "/usr/local/bin/kube-prompt")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		_log.Infow("unable to start pty", "error", err)
		return
	}

	wsrw := &wsReadWriter{conn}

	go func() {
		io.Copy(wsrw, f)
	}()

	go func() {
		io.Copy(f, wsrw)
	}()

	<-ctx.Done()
}

type wsReadWriter struct {
	conn *websocket.Conn
}

func (rw *wsReadWriter) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := rw.conn.NextReader()
		if err != nil {
			return 0, err
		}

		if msgType != websocket.TextMessage {
			continue
		}

		return reader.Read(p)
	}
}

func (rw *wsReadWriter) Write(p []byte) (n int, err error) {
	writer, err := rw.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	_log.Infow("writing", "message", string(p))
	return writer.Write(p)
}

var _ io.ReadWriter = (*wsReadWriter)(nil)
