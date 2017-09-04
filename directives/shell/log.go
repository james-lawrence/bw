package shell

import (
	"bufio"
	"bytes"
	"log"
)

func newLogging(l *log.Logger) logging {
	return logging{
		logger: l,
	}
}

// writes a buffer stream by line to the logs.
type logging struct {
	logger *log.Logger
}

func (t logging) Write(p []byte) (int, error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		t.logger.Print(scanner.Text())
	}

	return len(p), scanner.Err()
}
