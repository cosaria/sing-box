package stats

import (
	"testing"
)

func TestTrackerGetOrCreate(t *testing.T) {
	tr := NewTracker()
	s1 := tr.getOrCreate("ss-8388")
	s1.Upload.Add(1000)
	s1.Download.Add(2000)
	s2 := tr.getOrCreate("ss-8388")
	if s2.Upload.Load() != 1000 {
		t.Errorf("upload = %d, want 1000", s2.Upload.Load())
	}
	if s2.Download.Load() != 2000 {
		t.Errorf("download = %d, want 2000", s2.Download.Load())
	}
}

func TestTrackerSnapshot(t *testing.T) {
	tr := NewTracker()
	s1 := tr.getOrCreate("ss-8388")
	s1.Upload.Add(1000)
	s1.Download.Add(2000)
	s2 := tr.getOrCreate("vless-443")
	s2.Upload.Add(500)
	s2.Download.Add(1000)
	snap := tr.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot has %d entries, want 2", len(snap))
	}
	if snap["ss-8388"].Upload != 1000 || snap["ss-8388"].Download != 2000 {
		t.Errorf("ss-8388 = %+v, want {1000, 2000}", snap["ss-8388"])
	}
	if snap["vless-443"].Upload != 500 || snap["vless-443"].Download != 1000 {
		t.Errorf("vless-443 = %+v, want {500, 1000}", snap["vless-443"])
	}
}
