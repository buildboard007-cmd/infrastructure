package util

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetLogLevel(t *testing.T) {
	SetLogLevel(logrus.New(), "error")
	SetLogLevel(logrus.New(), "info")
	SetLogLevel(logrus.New(), "debug")
	SetLogLevel(logrus.New(), "other")
}
