package video

import (
	"testing"

	"github.com/pascalandy/ncly-video-convert/internal/publication"
)

// TestPublicationRecord makes the normal gate reject absent or outdated proof.
func TestPublicationRecord(t *testing.T) {
	if err := publication.CheckRecord("."); err != nil {
		t.Fatal(err)
	}
}
