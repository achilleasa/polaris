package log

import (
	"io"
	"os"

	"github.com/op/go-logging"
)

type Level logging.Level

// The levels that can be passed to the SetLevel function.
const (
	Debug Level = iota
	Info
	Notice
	Warning
	Error
)

// The logger format
var format = logging.MustStringFormatter(
	`%{color}[%{time:15:04:05.000}] [%{module}] [%{level}]%{color:reset} %{message}`,
)

// The internal leveled logger backend
var leveledBackend logging.LeveledBackend

// The logger interface
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Notice(v ...interface{})
	Noticef(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warning(v ...interface{})
	Warningf(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})
}

// Create a new named logger.
func New(name string) Logger {
	return logging.MustGetLogger(name)
}

// Override the backend output sink.
func SetSink(sink io.Writer) {
	backend := logging.NewLogBackend(sink, "", 0)
	backendWithFormatter := logging.NewBackendFormatter(backend, format)
	leveledBackend = logging.AddModuleLevel(backendWithFormatter)
	leveledBackend.SetLevel(logging.INFO, "")
	logging.SetBackend(leveledBackend)
}

// Set logger verbosity.
func SetLevel(level Level) {
	var loggerLevel logging.Level

	switch level {
	case Debug:
		loggerLevel = logging.DEBUG
	case Info:
		loggerLevel = logging.INFO
	case Notice:
		loggerLevel = logging.NOTICE
	case Warning:
		loggerLevel = logging.WARNING
	case Error:
		loggerLevel = logging.ERROR
	}

	leveledBackend.SetLevel(loggerLevel, "")
}

func init() {
	SetSink(os.Stdout)
	SetLevel(Notice)
}
