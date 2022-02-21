package debug

import (
	"io"
	"sync"
	"time"

	logv2 "github.com/RafaySystems/rafay-common/pkg/log/v2"
	"github.com/gorilla/websocket"
)

var (
	_log = logv2.GetLogger()
)

type wsReadWriter struct {
	conn *websocket.Conn
	m    sync.RWMutex
}

func newWSReadWriter(conn *websocket.Conn) io.ReadWriter {
	ws := &wsReadWriter{conn: conn}
	go ws.keepAlive(time.Second * 60)
	return ws
}

func (rw *wsReadWriter) Read(p []byte) (n int, err error) {

	for {
		var reader io.Reader
		_, reader, err = rw.conn.NextReader()
		if err != nil {
			_log.Errorw("unable to get next reader", "error", err)
			break
		}
		rw.conn.SetReadDeadline(time.Now().Add(time.Minute * 20))
		return reader.Read(p)
	}
	return 0, err
}

func (rw *wsReadWriter) Write(p []byte) (n int, err error) {

	rw.m.Lock()
	defer rw.m.Unlock()

	writer, err := rw.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		_log.Errorw("unable to get next writer", "error", err)
		return 0, err
	}
	defer writer.Close()
	rw.conn.SetReadDeadline(time.Now().Add(time.Minute * 20))
	_log.Debugw("writing", "message", string(p))

	return writer.Write(p)

}

func (rw *wsReadWriter) keepAlive(timeout time.Duration) {
	lastResponse := time.Now()
	rw.conn.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		errChan := make(chan error)
		ticker := time.NewTicker(timeout / 2)
		defer ticker.Stop()

	keepAliveLoop:
		for ; true; <-ticker.C {

			go func() {
				rw.m.Lock()
				defer rw.m.Unlock()
				err := rw.conn.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					errChan <- err

				}
			}()

			select {
			case <-errChan:
				break keepAliveLoop
			default:
				if time.Now().Sub(lastResponse) > timeout {
					rw.conn.Close()
					break keepAliveLoop
				}
			}
		}

	}()
}
