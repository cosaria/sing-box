package stats

import (
	"testing"

	"github.com/cosaria/sing-box/internal/store"
)

func TestCollectorFlush(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tracker := NewTracker()
	s := tracker.getOrCreate("ss-8388")
	s.Upload.Add(1000)
	s.Download.Add(2000)
	c := NewCollector(tracker, st)
	if err := c.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	summaries, err := st.GetTrafficSummary()
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Upload != 1000 {
		t.Errorf("upload = %d, want 1000", summaries[0].Upload)
	}
}

func TestCollectorFlushDelta(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tracker := NewTracker()
	s := tracker.getOrCreate("ss-8388")
	s.Upload.Add(1000)
	s.Download.Add(2000)
	c := NewCollector(tracker, st)
	c.Flush()
	s.Upload.Add(500)
	s.Download.Add(1000)
	c.Flush()
	summaries, _ := st.GetTrafficSummary()
	if summaries[0].Upload != 1500 {
		t.Errorf("total upload = %d, want 1500", summaries[0].Upload)
	}
	if summaries[0].Download != 3000 {
		t.Errorf("total download = %d, want 3000", summaries[0].Download)
	}
}
