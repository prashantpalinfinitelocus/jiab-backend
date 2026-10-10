package middlewares

import (
	"context"
	"github.com/sirupsen/logrus"
)

// LogRequestIDHook is a logrus hook that adds the request ID to the log entry
type LogRequestIDHook struct {
	Context context.Context
}

func (hook *LogRequestIDHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *LogRequestIDHook) Fire(entry *logrus.Entry) error {
	requestID := GetRequestIDFromContext(hook.Context)
	if requestID != "" {
		entry.Data["requestID"] = requestID
	}
	return nil
}
