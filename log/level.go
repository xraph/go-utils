package log

import "strings"

// LogLevel is the configuration-facing level type. It stays a string so that
// existing YAML, mapstructure and env bindings on LoggingConfig keep working.
type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
	LevelDebug LogLevel = "debug"
)

// level is the internal comparison type. It is an int8 so that the enabled
// check on the hot path is a single compare against an atomically loaded value.
type level int8

const (
	debugLevel level = iota - 1
	infoLevel
	warnLevel
	errorLevel
	fatalLevel
)

func (l level) String() string {
	switch l {
	case debugLevel:
		return "DEBUG"
	case infoLevel:
		return "INFO"
	case warnLevel:
		return "WARN"
	case errorLevel:
		return "ERROR"
	case fatalLevel:
		return "FATAL"
	default:
		return "INFO"
	}
}

// parseLevel maps a configured LogLevel onto the internal level. Unknown and
// empty values resolve to info, because a typo in a config file should not
// silence a service.
func parseLevel(l LogLevel) level {
	switch strings.ToLower(string(l)) {
	case "debug":
		return debugLevel
	case "warn", "warning":
		return warnLevel
	case "error":
		return errorLevel
	case "fatal":
		return fatalLevel
	default:
		return infoLevel
	}
}
