package log

import (
	"io"
	logPkg "log"
	"os"
)

// Expose the default sink so tests/becnhmarks can mute log output
var DefaultSink io.Writer = os.Stdout

const (
	LstdFlags = logPkg.LstdFlags
)

// The logger interface
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// Create a new logger using the default sink.
func New(prefix string, flags int) Logger {
	return logPkg.New(DefaultSink, prefix, flags)
}
