package log

import "testing"

func TestDeprecatedConstructorsStillWork(t *testing.T) {
	cases := map[string]func() Logger{
		"NewDevelopmentLogger":      NewDevelopmentLogger,
		"NewProductionLogger":       NewProductionLogger,
		"NewBeautifulLogger":        func() Logger { return NewBeautifulLogger("x") },
		"NewBeautifulLoggerCompact": func() Logger { return NewBeautifulLoggerCompact("x") },
		"NewBeautifulLoggerMinimal": func() Logger { return NewBeautifulLoggerMinimal("x") },
		"NewBeautifulLoggerJSON":    func() Logger { return NewBeautifulLoggerJSON("x") },
		"NewNoopLogger":             NewNoopLogger,
		"NewTestLogger":             NewTestLogger,
	}
	for name, ctor := range cases {
		t.Run(name, func(t *testing.T) {
			l := ctor()
			if l == nil {
				t.Fatal("returned nil")
			}
			// Must be safe to use immediately, including Sugar and Sync.
			l.Info("smoke", String("k", "v"))

			if l.Sugar() == nil {
				t.Error("Sugar() returned nil")
			}

			if err := l.Sync(); err != nil {
				t.Errorf("Sync() = %v", err)
			}
		})
	}
}

func TestNewBeautifulLoggerJSONActuallyEmitsJSON(t *testing.T) {
	// Finding 6 again, now through the deprecated name.
	// It writes to stderr, so this only checks that it is the JSON encoder.
	l, ok := NewBeautifulLoggerJSON("x").(*logger)
	if !ok {
		t.Fatalf("got %T, want *logger", NewBeautifulLoggerJSON("x"))
	}

	if _, ok := l.enc.(*jsonEncoder); !ok {
		t.Errorf("encoder is %T, want *jsonEncoder", l.enc)
	}
}
