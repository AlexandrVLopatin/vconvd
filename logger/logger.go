package logger

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
)

var Log = logging.MustGetLogger("vconvd")

type Config struct {
	LogFile  string
	LogLevel string
}

func SetupLogger(config Config) {
	backends := []logging.Backend{}

	stdBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stdFormatter := logging.NewBackendFormatter(stdBackend, logging.MustStringFormatter("%{color}%{time:2006/01/02 15:04:05.000} ▶ %{level:-8s} %{id:06x}%{color:reset} %{message}"))
	stdLevel := logging.AddModuleLevel(stdFormatter)
	logLevel, _ := logging.LogLevel(config.LogLevel)
	stdLevel.SetLevel(logLevel, "")
	backends = append(backends, stdLevel)

	if config.LogFile != "" {
		if logFile, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660); err == nil {
			fileBackend := logging.NewLogBackend(logFile, "", 0)
			fileFormatter := logging.NewBackendFormatter(fileBackend, logging.MustStringFormatter("%{time:2006/01/02 15:04:05.000} %{level:-8s} %{id:06x} %{message}"))
			fileLevel := logging.AddModuleLevel(fileFormatter)
			logLevel, _ := logging.LogLevel(config.LogLevel)
			fileLevel.SetLevel(logLevel, "")
			backends = append(backends, fileLevel)
		} else {
			print(fmt.Sprintf("Can not open log file: %v. Logging to stderr only.\n", err))
		}
	}

	logging.SetBackend(backends...)
}
