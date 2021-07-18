package shell

import (
	"bytes"
)

func newLogging(l logger) logging {
	return logging{
		logger: l,
		buf:    bytes.NewBufferString(""),
	}
}

// writes a buffer stream by line to the logs.
type logging struct {
	logger logger
	buf    *bytes.Buffer
}

func (t logging) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		switch i := bytes.IndexByte(p, '\n'); i {
		case -1:
			x, err := t.buf.Write(p)
			return n + x, err
		default:
			pre := p[:i+1]
			p = p[i+1:]

			if x, err := t.buf.Write(pre); err != nil {
				return n + x, err
			} else {
				n = n + x
			}

			t.logger.Print(t.buf.String())
			t.buf.Reset()
		}
	}

	return n, err
}
