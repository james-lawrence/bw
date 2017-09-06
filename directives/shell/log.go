package shell

import (
	"bufio"
	"bytes"
)

func newLogging(l logger) logging {
	return logging{
		logger: l,
	}
}

// writes a buffer stream by line to the logs.
type logging struct {
	logger logger
}

func (t logging) Write(p []byte) (int, error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		t.logger.Print(scanner.Text())
	}

	return len(p), scanner.Err()
}
