package tool

import (
	"bytes"
	"io"
)

// prefixWriter wraps an io.Writer and prepends a fixed prefix to every line.
type prefixWriter struct {
	w      io.Writer
	prefix []byte
	buf    []byte
}

func newPrefixWriter(prefix string, w io.Writer) io.Writer {
	return &prefixWriter{w: w, prefix: []byte(prefix)}
}

func (p *prefixWriter) Write(data []byte) (int, error) {
	p.buf = append(p.buf, data...)
	for {
		i := bytes.IndexByte(p.buf, '\n')
		if i < 0 {
			break
		}
		line := p.buf[:i+1]
		if _, err := p.w.Write(append(p.prefix, line...)); err != nil {
			return 0, err
		}
		p.buf = p.buf[i+1:]
	}
	return len(data), nil
}
