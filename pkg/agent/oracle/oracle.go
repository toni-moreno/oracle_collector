package oracle

import "github.com/sirupsen/logrus"

var (
	logDir string
	log    *logrus.Logger
)

func SetLogger(l *logrus.Logger) {
	log = l
}

// SetLogDir set log dir
func SetLogDir(l string) {
	logDir = l
}

func init() {
	OraList = NewInstanceList()
}
