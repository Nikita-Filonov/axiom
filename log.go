package axiom

// LogLevel classifies a structured Log for runtime sinks.
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

func (l LogLevel) String() string {
	return string(l)
}

// Log is a structured message emitted through Config.Log. Its level does not
// by itself change the Go test result; sinks choose how to handle it.
type Log struct {
	Text  string
	Level LogLevel
}

// LogOption configures a Log.
type LogOption func(*Log)

// NewLog returns a Log with the supplied options.
func NewLog(options ...LogOption) Log {
	l := Log{}
	for _, option := range options {
		option(&l)
	}
	return l
}

func WithLogText(text string) LogOption {
	return func(l *Log) { l.Text = text }
}

func WithLogLevel(level LogLevel) LogOption {
	return func(l *Log) { l.Level = level }
}

func NewDebugLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelDebug),
		WithLogText(text),
	)
}

func NewInfoLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelInfo),
		WithLogText(text),
	)
}

func NewWarningLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelWarning),
		WithLogText(text),
	)
}

func NewErrorLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelError),
		WithLogText(text),
	)
}

func NewFatalLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelFatal),
		WithLogText(text),
	)
}
