package native

import (
	"fmt"

	"github.com/0xAX/notificator"
	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/deployment/notifications"
	"github.com/james-lawrence/bw/internal/x/logx"
	"github.com/pkg/errors"
)

// New notifier
func New() *Notifier {
	notify := notificator.New(notificator.Options{
		DefaultIcon: "",
		AppName:     "bearded-wookie",
	})

	return &Notifier{
		dst:     notify,
		Title:   fmt.Sprintf("Deploy ${%s}", notifications.EnvDeployResult),
		Message: fmt.Sprintf("${%s}\nBy: ${%s}", notifications.EnvDeployID, notifications.EnvDeployInitiator),
	}
}

// Notifier - sends to the native notifications
type Notifier struct {
	Title   string
	Message string
	dst     *notificator.Notificator
}

// Notify send notification about a deploy
func (t Notifier) Notify(dc agent.DeployCommand) {
	title := notifications.ExpandEnv(t.Title, dc)
	msg := notifications.ExpandEnv(t.Message, dc)
	logx.MaybeLog(errors.Wrap(t.dst.Push(title, msg, "", urgency(dc.Command)), "failed to send notification"))
}

func urgency(c agent.DeployCommand_Command) string {
	switch c {
	case agent.DeployCommand_Done:
		return notificator.UR_NORMAL
	case agent.DeployCommand_Cancel:
		return notificator.UR_NORMAL
	default:
		return notificator.UR_CRITICAL
	}
}
