package video

import (
	"os"
	"path/filepath"
	"testing"
)

// The runtime warning needs a build with Quick Sync that lists no qsv
// encoder at all, so a GPU that encodes H.264 but not H.265 gets none.
func TestQuickSyncMissing(t *testing.T) {
	offers := "   x264\n   x265_10bit\n   Use 'qsv' to enable QSV decoding\n"
	for _, c := range []struct {
		name string
		help string
		want bool
	}{
		{"no qsv encoder", offers, true},
		{"qsv for H.264 only", offers + "   qsv_h264\n", false},
		{"build without Quick Sync", "   x264\n   x265_10bit\n", false},
	} {
		if got := quickSyncMissing(c.help); got != c.want {
			t.Errorf("%s: quickSyncMissing = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIntelGPU(t *testing.T) {
	sys := t.TempDir()
	vendor := func(card, id string) {
		dir := filepath.Join(sys, "class", "drm", card, "device")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "vendor"), []byte(id+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vendor("card0", "0x10de")
	if intelGPU(sys) {
		t.Error("an NVIDIA card reads as Intel")
	}
	vendor("card1", "0x8086")
	if !intelGPU(sys) {
		t.Error("an Intel card reads as another vendor")
	}
}

// HandBrake reads a PNG as 0 ticks and a JPEG as one frame at 25 fps. Its
// frame rate counts a 27 MHz clock, and ticks count 90 kHz.
func TestStill(t *testing.T) {
	const num, den = 27000000, 1080000
	for _, c := range []struct {
		name     string
		ticks    uint64
		num, den int64
		want     bool
	}{
		{"PNG", 0, num, den, true},
		{"JPEG, one frame", 3600, num, den, true},
		{"just over one frame", 3601, num, den, false},
		{"three frames", 10800, num, den, false},
		{"no frame rate, no tick", 0, 0, 0, true},
		{"no frame rate, one tick", 1, 0, 0, false},
		{"negative frame rate", 0, -num, den, true},
		{"a product past 64 bits", 1 << 62, num, den, false},
	} {
		title := title{Duration: &struct{ Ticks uint64 }{c.ticks}}
		title.FrameRate.Num, title.FrameRate.Den = c.num, c.den
		if got := title.still(); got != c.want {
			t.Errorf("%s: still = %v, want %v", c.name, got, c.want)
		}
	}
}
