package log

// ANSI color codes and styles for terminal output.
const (
	// Reset and basic colors.
	Reset = "\033[0m"

	// Text styles.
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Foreground colors.
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors.
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors.
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// ColorScheme defines colors for different parts of a log entry.
type ColorScheme struct {
	Level     string
	Timestamp string
	Message   string
	Fields    string
	Key       string
	Value     string
}

// Predefined color schemes for each log level.
var (
	DebugColors = ColorScheme{
		Level:     Dim + Cyan,
		Timestamp: Dim + BrightBlack,
		Message:   Cyan,
		Fields:    Dim + Cyan,
		Key:       BrightCyan,
		Value:     Cyan,
	}

	InfoColors = ColorScheme{
		Level:     Bold + Green,
		Timestamp: Dim + BrightBlack,
		Message:   White,
		Fields:    Green,
		Key:       BrightGreen,
		Value:     Green,
	}

	WarnColors = ColorScheme{
		Level:     Bold + Yellow,
		Timestamp: Dim + BrightBlack,
		Message:   Yellow,
		Fields:    Yellow,
		Key:       BrightYellow,
		Value:     Yellow,
	}

	ErrorColors = ColorScheme{
		Level:     Bold + Red,
		Timestamp: Dim + BrightBlack,
		Message:   BrightRed,
		Fields:    Red,
		Key:       BrightRed,
		Value:     Red,
	}

	FatalColors = ColorScheme{
		Level:     Bold + BgRed + White,
		Timestamp: Dim + BrightBlack,
		Message:   Bold + Magenta,
		Fields:    Magenta,
		Key:       BrightMagenta,
		Value:     Magenta,
	}
)
