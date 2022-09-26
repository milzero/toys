package common

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewLog() *logrus.Logger {
	logger := &lumberjack.Logger{
		Filename:   "mini.log",
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	log := logrus.New()

	mw := io.MultiWriter(os.Stdout, logger)
	log.SetOutput(mw)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:               false,
		DisableColors:             false,
		ForceQuote:                true,
		DisableQuote:              false,
		EnvironmentOverrideColors: false,
		DisableTimestamp:          false,
		FullTimestamp:             false,
		TimestampFormat:           "",
		DisableSorting:            false,
		SortingFunc:               nil,
		DisableLevelTruncation:    false,
		PadLevelText:              false,
		QuoteEmptyFields:          false,
		FieldMap:                  nil,
		CallerPrettyfier:          nil,
	})
	log.SetReportCaller(true)
	log.SetLevel(logrus.DebugLevel)

	return log
}
