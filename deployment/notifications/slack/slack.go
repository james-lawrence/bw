package slack

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

type field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type notification struct {
	Channel string  `json:"channel"`
	Emoji   string  `json:"icon_emoji"`
	Text    string  `json:"text"`
	Fields  []field `json:"fields"`
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

	msg := os.ExpandEnv(t.Message)

	n := notification{
		Channel: t.Channel,
		Text:    msg,
		Fields: []field{
			{
				Title: bw.RandomID(dc.Archive.DeploymentID).String(),
				Value: dc.Command.String(),
			},
		},
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
