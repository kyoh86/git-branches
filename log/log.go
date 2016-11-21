package log

import (
	"os"

	"github.com/Sirupsen/logrus"
)

// InitLogger : ログの初期化
func InitLogger() {
	formatter := logrus.TextFormatter{
		DisableTimestamp: true, // bool
	}
	logrus.SetOutput(os.Stderr)
	logrus.SetFormatter(&formatter)
	initLevel()
}
