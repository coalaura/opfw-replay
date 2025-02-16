package main

import (
	"bytes"
	"errors"
	"io"
)

type WriterSeeker struct {
	pos int
	buf bytes.Buffer
}

func NewWriteSeeker(size int) *WriterSeeker {
	return &WriterSeeker{
		pos: 0,
		buf: *bytes.NewBuffer(make([]byte, 0, size)),
	}
}

func (ws *WriterSeeker) Write(p []byte) (n int, err error) {
	if extra := ws.pos - ws.buf.Len(); extra > 0 {
		if _, err := ws.buf.Write(make([]byte, extra)); err != nil {
			return n, err
		}
	}

	if ws.pos < ws.buf.Len() {
		n = copy(ws.buf.Bytes()[ws.pos:], p)
		p = p[n:]
	}

	if len(p) > 0 {
		var bn int

		bn, err = ws.buf.Write(p)
		n += bn
	}

	ws.pos += n

	return n, err
}

func (ws *WriterSeeker) Seek(offset int64, whence int) (int64, error) {
	newPos, offs := 0, int(offset)

	switch whence {
	case io.SeekStart:
		newPos = offs
	case io.SeekCurrent:
		newPos = ws.pos + offs
	case io.SeekEnd:
		newPos = ws.buf.Len() + offs
	}

	if newPos < 0 {
		return 0, errors.New("negative result pos")
	}

	ws.pos = newPos

	return int64(newPos), nil
}

func (ws *WriterSeeker) Bytes() []byte {
	return ws.buf.Bytes()
}

func (ws *WriterSeeker) Close() error {
	return nil
}
