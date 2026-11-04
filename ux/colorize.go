package ux

import (
	"fmt"

	"github.com/james-lawrence/bw/agent"
	"github.com/james-lawrence/bw/internal/langx"
	"github.com/logrusorgru/aurora"
)

const defaultFormatPattern = "%[1]s - %-5[2]s - %[3]s"

func typographyunexpectmsg(au aurora.Aurora, m *agent.Message) aurora.Value {
	return typographymsg(au, m, agent.LogMessage(m, agent.Log_DEBUG, m.Type.String()))
}

func typographymsg(au aurora.Aurora, m *agent.Message, evt *agent.Message_Log) aurora.Value {
	return typography(au, defaultFormatPattern, m, evt)
}

func typography(au aurora.Aurora, pattern string, m *agent.Message, evt *agent.Message_Log) aurora.Value {
	printlevel := langx.FirstNonZero(evt.Log.Level, agent.Log_INFO)
	msg := fmt.Sprintf(pattern, messagePrefix(m), printlevel, evt.Log.Log)
	switch agent.Log_Level(evt.Log.Level) {
	case agent.Log_DEBUG:
		return au.Gray(11, msg)
	case agent.Log_WARN:
		return au.Yellow(msg)
	case agent.Log_ERROR:
		return au.Red(msg)
	case agent.Log_USER:
		return au.Green(msg)
	default:
		return au.Reset(msg)
	}
}
