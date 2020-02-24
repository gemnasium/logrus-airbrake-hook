package airbrake // import "gopkg.in/gemnasium/logrus-airbrake-hook.v4"

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/airbrake/gobrake/v4"
	"github.com/sirupsen/logrus"
)

// default levels to be fired when logging on
var defaultLevels = []logrus.Level{
	logrus.ErrorLevel,
	logrus.FatalLevel,
	logrus.PanicLevel,
}

// AirbrakeHook to send exceptions to an exception-tracking service compatible
// with the Airbrake API.
type airbrakeHook struct {
	Airbrake *gobrake.Notifier
	levels   []logrus.Level
}

func NewHook(projectID int64, apiKey, env string, levels ...logrus.Level) *airbrakeHook {
	airbrake := gobrake.NewNotifier(projectID, apiKey)
	airbrake.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
		if env == "development" {
			return nil
		}
		notice.Context["environment"] = env
		return notice
	})

	if len(levels) == 0 {
		levels = defaultLevels
	}

	hook := &airbrakeHook{
		Airbrake: airbrake,
		levels:   levels,
	}
	return hook
}

func (hook *airbrakeHook) Fire(entry *logrus.Entry) error {
	var notifyErr error
	err, ok := entry.Data["error"].(error)
	if ok {
		notifyErr = err
	} else {
		notifyErr = errors.New(entry.Message)
	}
	var req *http.Request
	for k, v := range entry.Data {
		if r, ok := v.(*http.Request); ok {
			req = r
			delete(entry.Data, k)
			break
		}
	}
	notice := hook.Airbrake.Notice(notifyErr, req, 3)
	for k, v := range entry.Data {
		notice.Context[k] = fmt.Sprintf("%s", v)
	}

	hook.sendNotice(notice)
	return nil
}

func (hook *airbrakeHook) sendNotice(notice *gobrake.Notice) {
	if _, err := hook.Airbrake.SendNotice(notice); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send error to Airbrake: %v\n", err)
	}
}

func (hook *airbrakeHook) Levels() []logrus.Level {
	return hook.levels
}
