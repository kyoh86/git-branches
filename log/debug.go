// +build debug

package log

import "github.com/Sirupsen/logrus"

func initLevel() {
	logrus.SetLevel(logrus.DebugLevel)
}

func verbose() {
}
