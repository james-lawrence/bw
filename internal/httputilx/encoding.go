package httputilx

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// EncodeJSON encode data into the http.Request body.
func EncodeJSON(req *http.Request, body interface{}) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = json.Marshal(body); err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = ioutil.NopCloser(bytes.NewReader(encoded))

	return nil
}
