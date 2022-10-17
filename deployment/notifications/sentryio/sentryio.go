package sentryio

// https://docs.sentry.io/api/releases/create-a-new-deploy-for-an-organization/
// curl https://sentry.io/api/0/organizations/{organization_slug}/releases/{version}/deploys/ \
//  -H 'Authorization: Bearer <auth_token>' \
//  -H 'Content-Type: application/json' \
//  -d '{"environment":"prod"}'
//
//
import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/james-lawrence/bw"
	"github.com/james-lawrence/bw/agent"
	"github.com/pkg/errors"
)

type notification struct {
	Version      string    `json:"version"`
	Commit       string    `json:"ref"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Projects     []string  `json:"projects"`
	DateReleased time.Time `json:"dateReleased"`
}

// New ...
func New() *Notifier {
	return &Notifier{
		Webhook: "https://sentry.io/api/0/organizations/{organization}/releases/",
		client:  defaultClient(),
	}
}

func defaultClient() *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
	}
}

// Notifier - creates a release with sentry.io.
type Notifier struct {
	Environment   string
	Name          string
	URL           string
	Projects      []string
	Organization  string // optional
	Authorization string
	Webhook       string
	client        *http.Client
}

// Notify send notification about a deploy
func (t Notifier) Notify(dc *agent.DeployCommand) {
	var (
		err  error
		raw  []byte
		req  *http.Request
		resp *http.Response
	)

	switch dc.Command {
	case agent.DeployCommand_Done:
	default:
		return // nothing to do if not completed
	}

	n := notification{
		Version:      bw.RandomID(dc.Archive.DeploymentID).String(),
		Name:         t.Name,
		URL:          t.URL,
		Projects:     t.Projects,
		DateReleased: time.Unix(dc.Archive.Dts, 0),
		Commit:       dc.Archive.Commit,
	}

	if raw, err = json.Marshal(n); err != nil {
		log.Println(errors.Wrap(err, "failed to encode slack notification"))
		return
	}

	webhook := t.Webhook
	webhook = strings.ReplaceAll(webhook, "{organization}", t.Organization)

	if req, err = http.NewRequest(http.MethodPost, webhook, bytes.NewReader(raw)); err != nil {
		log.Println(errors.Wrap(err, "failed to create request"))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Authorization))

	if resp, err = t.client.Do(req); err != nil {
		log.Println(errors.Wrap(err, "failed to post webhook"))
		return
	}

	if resp.StatusCode > 299 {
		log.Println("webhook request failed with status code", resp.StatusCode)
	}
}
