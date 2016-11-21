// +build !debug

package log

import "github.com/Sirupsen/logrus"

func initLevel() {
	logrus.SetLevel(logrus.WarnLevel)
}

func verbose() {
	logrus.SetLevel(logrus.InfoLevel)
}
