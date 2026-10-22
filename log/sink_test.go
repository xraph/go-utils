package log

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// kv is one field copied out of a Record, so a test can read it after Emit
// has returned and the logger has recycled the slice.
type kv struct {
	key string
	val any
}

// recorded is a Record whose Fields have been deep-copied.
type recorded struct {
	Record

	kvs []kv
}

type recSink struct {
	mu   sync.Mutex
	recs []recorded
}

func (s *recSink) Emit(r Record) {
	c := recorded{Record: r}
	c.Fields = nil

	for _, f := range r.Fields {
		c.kvs = append(c.kvs, kv{key: f.Key(), val: f.Value()})
	}

	s.mu.Lock()
	s.recs = append(s.recs, c)
	s.mu.Unlock()
}

func (s *recSink) snapshot() []recorded {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]recorded(nil), s.recs...)
}

type panicSink struct{}

func (panicSink) Emit(Record) { panic("sink exploded") }

func sinkTestLogger(lv LogLevel) Logger {
	return New(Config{Format: FormatJSON, Output: io.Discard, Level: lv})
}

func mustAttach(t *testing.T, l Logger, s Sink) func() {
	t.Helper()

	detach, err := AttachSink(l, s)
	if err != nil {
		t.Fatalf("AttachSink: %v", err)
	}

	return detach
}

func TestAttachSinkReachesExistingAndNewChildren(t *testing.T) {
	root := sinkTestLogger(LevelDebug)
	child := root.With(String("a", "1")).Named("svc")

	s := &recSink{}
	mustAttach(t, root, s)

	later := root.Named("x")

	child.Info("from child")
	later.Info("from later")

	recs := s.snapshot()
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}

	if recs[0].Logger != "svc" {
		t.Errorf("child Logger = %q, want svc", recs[0].Logger)
	}

	if len(recs[0].kvs) != 1 || recs[0].kvs[0] != (kv{"a", "1"}) {
		t.Errorf("child fields = %v, want [a=1]", recs[0].kvs)
	}

	if recs[1].Logger != "x" {
		t.Errorf("later Logger = %q, want x", recs[1].Logger)
	}
}

func TestDetachStopsDelivery(t *testing.T) {
	l := sinkTestLogger(LevelDebug)
	s := &recSink{}
	detach := mustAttach(t, l, s)

	l.Info("one")
	detach()
	l.Info("two")
	detach()

	if got := len(s.snapshot()); got != 1 {
		t.Fatalf("got %d records, want 1", got)
	}
}

func TestSinkSeesFieldsInEncoderOrder(t *testing.T) {
	root := sinkTestLogger(LevelDebug)
	s := &recSink{}
	mustAttach(t, root, s)

	root.With(String("k", "logger")).Info("m", String("k", "call"), Int("n", 2))

	recs := s.snapshot()
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}

	r := recs[0]

	want := []kv{{"k", "logger"}, {"k", "call"}, {"n", int64(2)}}
	if len(r.kvs) != len(want) {
		t.Fatalf("fields = %v, want %v", r.kvs, want)
	}

	for i := range want {
		if r.kvs[i] != want[i] {
			t.Errorf("field %d = %v, want %v", i, r.kvs[i], want[i])
		}
	}

	if r.Message != "m" {
		t.Errorf("Message = %q, want m", r.Message)
	}

	if r.Level != LevelInfo {
		t.Errorf("Level = %q, want %q", r.Level, LevelInfo)
	}

	if r.Time.IsZero() {
		t.Error("Time is zero")
	}
}

func TestSinkNeverSeesFilteredLevel(t *testing.T) {
	l := sinkTestLogger(LevelWarn)
	s := &recSink{}
	mustAttach(t, l, s)

	l.Info("dropped")
	l.Warn("kept")

	recs := s.snapshot()
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}

	if recs[0].Level != LevelWarn {
		t.Errorf("Level = %q, want %q", recs[0].Level, LevelWarn)
	}
}

func TestSinkLevelMapping(t *testing.T) {
	l := sinkTestLogger(LevelDebug)
	s := &recSink{}
	mustAttach(t, l, s)

	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")

	want := []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError}

	recs := s.snapshot()
	if len(recs) != len(want) {
		t.Fatalf("got %d records, want %d", len(recs), len(want))
	}

	for i, w := range want {
		if recs[i].Level != w {
			t.Errorf("record %d Level = %q, want %q", i, recs[i].Level, w)
		}
	}

	if got := fatalLevel.logLevel(); got != LevelFatal {
		t.Errorf("fatalLevel.logLevel() = %q, want %q", got, LevelFatal)
	}
}

func TestPanickingSinkDoesNotPanicCaller(t *testing.T) {
	l := sinkTestLogger(LevelDebug)
	s := &recSink{}

	mustAttach(t, l, panicSink{})
	mustAttach(t, l, s)

	l.Info("survives")

	if got := len(s.snapshot()); got != 1 {
		t.Fatalf("got %d records, want 1", got)
	}
}

func TestAttachSinkUnsupported(t *testing.T) {
	s := &recSink{}

	cases := map[string]Logger{
		"noop":    NewNoopLogger(),
		"capture": NewTestLogger(),
		"foreign": struct{ Logger }{},
	}

	for name, l := range cases {
		t.Run(name, func(t *testing.T) {
			detach, err := AttachSink(l, s)
			if !errors.Is(err, ErrSinksUnsupported) {
				t.Fatalf("err = %v, want ErrSinksUnsupported", err)
			}

			if detach == nil {
				t.Fatal("detach is nil")
			}

			detach()
		})
	}
}

func TestCallerIsPassedWhenEnabled(t *testing.T) {
	l := New(Config{Format: FormatJSON, Output: io.Discard, Level: LevelDebug, AddCaller: true})
	s := &recSink{}
	mustAttach(t, l, s)

	l.Info("where")

	recs := s.snapshot()
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}

	if !strings.Contains(recs[0].Caller, "sink_test.go:") {
		t.Errorf("Caller = %q, want it to name sink_test.go", recs[0].Caller)
	}
}

func TestCallerIsEmptyWhenDisabled(t *testing.T) {
	l := sinkTestLogger(LevelDebug)
	s := &recSink{}
	mustAttach(t, l, s)

	l.Info("where")

	if c := s.snapshot()[0].Caller; c != "" {
		t.Errorf("Caller = %q, want empty", c)
	}
}

func TestSinkConcurrentAttachAndLog(t *testing.T) {
	root := sinkTestLogger(LevelDebug)

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			l := root.With(String("g", "x")).Named("w")
			for range 1000 {
				l.Info("line", Int("i", 1))
			}
		})
	}

	wg.Go(func() {
		for range 100 {
			detach, err := AttachSink(root, &recSink{})
			if err != nil {
				t.Errorf("AttachSink: %v", err)

				return
			}

			detach()
		}
	})

	wg.Wait()
}

// TestNoSinkPathAllocatesNothingExtra pins the cost of the feature when nobody
// uses it: a line written with no sink attached costs no allocations.
func TestNoSinkPathAllocatesNothingExtra(t *testing.T) {
	skipUnderRace(t)

	l := sinkTestLogger(LevelDebug).With(String("a", "1"))

	got := testing.AllocsPerRun(200, func() { l.Info("m") })
	if got != 0 {
		t.Errorf("allocs per line with no sink = %v, want 0", got)
	}
}
