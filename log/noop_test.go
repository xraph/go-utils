package log

import (
	"context"
	"testing"
)

func TestNoopLoggerSatisfiesInterface(t *testing.T) {
	var l = NewNoopLogger()
	l.Debug("d", String("k", "v"))
	l.Info("i", String("k", "v"))
	l.Warn("w", String("k", "v"))
	l.Error("e", String("k", "v"))
	l.Debugf("%s", "x")
	l.Infof("%s", "x")
	l.Warnf("%s", "x")
	l.Errorf("%s", "x")

	if l.With(String("a", "b")) == nil {
		t.Error("With returned nil")
	}

	if l.WithContext(context.Background()) == nil {
		t.Error("WithContext returned nil")
	}

	if l.Named("child") == nil {
		t.Error("Named returned nil")
	}

	if l.Sugar() == nil {
		t.Error("Sugar returned nil; a nil SugarLogger panics on first use")
	}

	if err := l.Sync(); err != nil {
		t.Errorf("Sync() = %v, want nil", err)
	}
}

func TestNoopSugarLoggerIsUsable(t *testing.T) {
	s := NewNoopLogger().Sugar()
	s.Debugw("d", "k", "v")
	s.Infow("i", "k", "v")
	s.Warnw("w", "k", "v")
	s.Errorw("e", "k", "v")

	if s.With("k", "v") == nil {
		t.Error("SugarLogger.With returned nil")
	}
}

func TestNoopLoggerDoesNotAllocate(t *testing.T) {
	skipUnderRace(t)

	// This is the reason noop is a distinct type rather than "a logger with a
	// very high level": empty method bodies inline, and the variadic slice
	// does not escape.
	l := NewNoopLogger()

	got := testing.AllocsPerRun(200, func() {
		l.Info("message", String("k", "v"), Int("n", 1))
	})
	if got != 0 {
		t.Errorf("noop Info allocated %.0f times, want 0", got)
	}
}
