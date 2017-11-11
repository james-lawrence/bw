package deployment

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/james-lawrence/bw"
	"github.com/pkg/errors"
)

type dlog struct {
	uid string
	log *log.Logger
}

func (t dlog) Print(v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprint(v...)))
}

func (t dlog) Printf(format string, v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprintf(format, v...)))
}

func (t dlog) Println(v ...interface{}) {
	t.log.Output(2, fmt.Sprintf("%s: %s", t.uid, fmt.Sprintln(v...)))
}

func newLogger(uid bw.RandomID, root, prefix string) (dst *os.File, _dlog dlog, err error) {
	if dst, err = os.OpenFile(filepath.Join(root, "deploy.log"), os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0600); err != nil {
		return nil, _dlog, errors.WithStack(err)
	}

	return dst, dlog{uid: uid.String(), log: log.New(io.MultiWriter(os.Stderr, dst), prefix, log.Flags()^log.Lshortfile)}, nil
}

func logErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
