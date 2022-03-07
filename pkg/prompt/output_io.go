package prompt

import (
	"io"
)

type ioWriter struct {
	VT100Writer
	w io.Writer
}

// Flush to flush buffer.
func (w *ioWriter) Flush() error {
	//_log.Infow("before flush", "message", string(w.buffer))
	_, err := w.w.Write(w.buffer)
	if err != nil {
		_log.Infow("unable to flush", "error", err)
		return err
	}
	w.buffer = []byte{}
	return nil
}

var _ ConsoleWriter = (*ioWriter)(nil)

// NewIOWriter returns new console writer which writes to io.Writer
func NewIOWriter(w io.Writer) ConsoleWriter {
	return &ioWriter{w: w}
}
