package log

import "testing"

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in   LogLevel
		want level
	}{
		{"debug", debugLevel},
		{"DEBUG", debugLevel},
		{"info", infoLevel},
		{"warn", warnLevel},
		{"warning", warnLevel},
		{"error", errorLevel},
		{"fatal", fatalLevel},
		{"", infoLevel},         // empty config defaults to info
		{"nonsense", infoLevel}, // unknown defaults to info
	}
	for _, c := range cases {
		if got := parseLevel(c.in); got != c.want {
			t.Errorf("parseLevel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestLevelOrdering(t *testing.T) {
	if debugLevel >= infoLevel || infoLevel >= warnLevel ||
		warnLevel >= errorLevel || errorLevel >= fatalLevel {
		t.Fatal("levels must be strictly increasing in severity")
	}
}

func TestLevelString(t *testing.T) {
	cases := map[level]string{
		debugLevel: "DEBUG",
		infoLevel:  "INFO",
		warnLevel:  "WARN",
		errorLevel: "ERROR",
		fatalLevel: "FATAL",
	}
	for lv, want := range cases {
		if got := lv.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", lv, got, want)
		}
	}
}
