package log

import (
	"context"
	"sync"
	"testing"
)

func TestCaptureLoggerRecordsFields(t *testing.T) {
	l := NewTestLogger().(*TestLogger)

	l.Info("request", String("method", "GET"), Int("status", 200))

	logs := l.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("got %d entries, want 1", len(logs))
	}

	if logs[0].Level != "INFO" || logs[0].Message != "request" {
		t.Errorf("entry = %+v", logs[0])
	}

	method, ok := logs[0].Field("method")
	if !ok || method != "GET" {
		t.Errorf("Field(method) = %v, %v; want GET, true", method, ok)
	}

	status, ok := logs[0].Field("status")
	if !ok || status != int64(200) {
		t.Errorf("Field(status) = %v, %v; want 200, true", status, ok)
	}

	if _, ok := logs[0].Field("absent"); ok {
		t.Error("Field(absent) reported present")
	}
}

// The old With() returned the receiver and dropped the fields entirely.
func TestCaptureLoggerWithRetainsFields(t *testing.T) {
	l := NewTestLogger().(*TestLogger)

	l.With(String("service", "api")).Info("msg", String("k", "v"))

	logs := l.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("got %d entries, want 1", len(logs))
	}

	if svc, ok := logs[0].Field("service"); !ok || svc != "api" {
		t.Errorf("With() field was lost: %+v", logs[0].Fields)
	}

	if v, ok := logs[0].Field("k"); !ok || v != "v" {
		t.Errorf("call field was lost: %+v", logs[0].Fields)
	}
}

func TestCaptureLoggerFieldOrderIsPreserved(t *testing.T) {
	l := NewTestLogger().(*TestLogger)
	l.Info("msg", String("a", "1"), String("b", "2"), String("c", "3"))

	got := l.GetLogs()[0].Fields

	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].Key() != want[i] {
			t.Errorf("field %d = %q, want %q", i, got[i].Key(), want[i])
		}
	}
}

func TestCaptureLoggerSugarIsNotNil(t *testing.T) {
	// The old TestLogger returned nil from Sugar(), which panicked on use.
	s := NewTestLogger().Sugar()
	if s == nil {
		t.Fatal("Sugar() returned nil")
	}

	s.Infow("sugar", "k", "v")
}

func TestCaptureLoggerNamedAndContext(t *testing.T) {
	l := NewTestLogger().(*TestLogger)
	ctx := WithRequestID(context.Background(), "req1")

	l.Named("child").WithContext(ctx).Info("msg")

	logs := l.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("got %d entries, want 1", len(logs))
	}

	if id, ok := logs[0].Field("request_id"); !ok || id != "req1" {
		t.Errorf("context field was lost: %+v", logs[0].Fields)
	}
}

// A parent and its With/Named children all append to one shared buffer, so the
// lock guarding it has to be shared too. With a per-clone mutex this fails
// under -race immediately.
func TestCaptureLoggerIsSafeAcrossClones(t *testing.T) {
	root := NewTestLogger().(*TestLogger)

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			// Half log through the root, half through a fresh clone.
			if i%2 == 0 {
				root.Info("from root", Int("i", i))
			} else {
				root.With(String("clone", "yes")).Named("child").Info("from clone", Int("i", i))
			}
		}(i)
	}

	wg.Wait()

	if got := len(root.GetLogs()); got != goroutines {
		t.Errorf("captured %d entries, want %d", got, goroutines)
	}
}

func TestCaptureLoggerHelpers(t *testing.T) {
	l := NewTestLogger().(*TestLogger)
	l.Info("one")
	l.Error("two")
	l.Error("three")

	if got := l.CountLogs("ERROR"); got != 2 {
		t.Errorf("CountLogs(ERROR) = %d, want 2", got)
	}

	if !l.AssertHasLog("INFO", "one") {
		t.Error("AssertHasLog(INFO, one) = false")
	}

	if l.AssertHasLog("INFO", "missing") {
		t.Error("AssertHasLog matched a message that was never logged")
	}

	if got := len(l.GetLogsByLevel("ERROR")); got != 2 {
		t.Errorf("GetLogsByLevel(ERROR) returned %d, want 2", got)
	}

	l.Clear()

	if got := len(l.GetLogs()); got != 0 {
		t.Errorf("Clear left %d entries", got)
	}
}

// Finding 3 again, for the capture logger specifically. This assertion
// lives here rather than in the config task's file so that each commit's
// test binary compiles on its own.
func TestSetGlobalLoggerAcceptsTheCaptureLogger(t *testing.T) {
	original := GetGlobalLogger()

	t.Cleanup(func() { SetGlobalLogger(original) })

	want := NewTestLogger()
	SetGlobalLogger(want)

	if got := GetGlobalLogger(); got != want {
		t.Errorf("SetGlobalLogger(%T) then GetGlobalLogger() returned %T, want the same value", want, got)
	}
}
