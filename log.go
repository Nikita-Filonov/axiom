package axiom

// LogLevel classifies a structured Log for runtime sinks.
type LogLevel string

// LogLevel values classify structured log messages.
const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// String returns the log level name.
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

// WithLogText sets the structured log message.
func WithLogText(text string) LogOption {
	return func(l *Log) { l.Text = text }
}

// WithLogLevel sets the level dispatched to log sinks.
func WithLogLevel(level LogLevel) LogOption {
	return func(l *Log) { l.Level = level }
}

// NewDebugLog creates a debug-level log message.
func NewDebugLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelDebug),
		WithLogText(text),
	)
}

// NewInfoLog creates an info-level log message.
func NewInfoLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelInfo),
		WithLogText(text),
	)
}

// NewWarningLog creates a warning-level log message.
func NewWarningLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelWarning),
		WithLogText(text),
	)
}

// NewErrorLog creates an error-level log message.
func NewErrorLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelError),
		WithLogText(text),
	)
}

// NewFatalLog creates a fatal-level log message.
func NewFatalLog(text string) Log {
	return NewLog(
		WithLogLevel(LogLevelFatal),
		WithLogText(text),
	)
}
