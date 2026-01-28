package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func New(level string) (*logrus.Logger, error) {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	log.SetLevel(lvl)

	return log, nil
}
