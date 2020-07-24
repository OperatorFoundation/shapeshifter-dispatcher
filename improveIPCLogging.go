package main

import (
	"errors"
	"github.com/OperatorFoundation/shapeshifter-dispatcher/common/log"
)

func validateIPCLogLevel(ipcLogLevel string) (int, error) {
	switch ipcLogLevel {
	case "NONE":
		return log.LevelNone, nil

	case "ERROR":
		return log.LevelError, nil

	case "WARN":
		return log.LevelWarn, nil

	case "INFO":
		return log.LevelInfo, nil

	case "DEBUG":
		return log.LevelDebug, nil

	default:
		return -1, errors.New("invalid log level")
	}
}

func ipcLogMessage(logLevel int, message string) {
	var logLevelStr string
	switch logLevel {
	case log.LevelNone:
		logLevelStr = "NONE"
	case log.LevelError:
		logLevelStr = "ERROR"
	case log.LevelWarn:
		logLevelStr = "WARN"
	case log.LevelInfo:
		logLevelStr = "INFO"
	case log.LevelDebug:
		logLevelStr = "DEBUG"
	default:
		return
	}
	println("LOG "+logLevelStr+" "+message)
}
