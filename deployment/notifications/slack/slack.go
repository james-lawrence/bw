package slack

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment/notifications"
	"github.com/pkg/errors"
)

type notification struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// New ...
func New() *Notifier {
	return &Notifier{
		client: defaultClient(),
	}
}

func defaultClient() *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
	}
}

// Notifier - sends an message to a slack webhook.
type Notifier struct {
	Channel string
	Webhook string
	Message string
	client  *http.Client
}

// Notify send notification about a deploy
func (t Notifier) Notify(dc agent.DeployCommand) {
	var (
		err  error
		raw  []byte
		resp *http.Response
	)

	msg := notifications.ExpandEnv(t.Message, dc)

	n := notification{
		Channel: t.Channel,
		Text:    msg,
	}

	if raw, err = json.Marshal(n); err != nil {
		log.Println(errors.Wrap(err, "failed to encode slack notification"))
		return
	}

	if resp, err = http.Post(t.Webhook, "application/json", bytes.NewReader(raw)); err != nil {
		log.Println(errors.Wrap(err, "failed to post webhook"))
		return
	}

	if resp.StatusCode > 299 {
		log.Println("webhook request failed with status code", resp.StatusCode)
	}
}

func colorFromCommand(c agent.DeployCommand_Command) string {
	const (
		ok     = "#8eb573"
		failed = "#FF0000"
		warn   = "#00FFFF"
	)

	switch c {
	case agent.DeployCommand_Done:
		return ok
	case agent.DeployCommand_Cancel:
		return warn
	default:
		return failed
	}
}
