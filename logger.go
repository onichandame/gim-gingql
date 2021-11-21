package gimgingql

import "github.com/sirupsen/logrus"

func newLogger() *logrus.Entry {
	logger := logrus.WithFields(logrus.Fields{
		"pkg": "Gim/GinGraphql",
	})
	logger.Logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	return logger
}
