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
