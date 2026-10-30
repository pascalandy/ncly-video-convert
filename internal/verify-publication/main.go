// verify-publication runs the expensive proof in a disposable source snapshot.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pascalandy/ncly-video-convert/internal/publication"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Println("Usage: go run ./internal/verify-publication\nRun on Linux from the repository root. Requires Go, Git, mise, and network for pinned tools.\nKeeps scratch logs under TMPDIR and writes verification/publication/record.json only after all checks pass.")
		return
	}
	var err error
	if len(os.Args) != 1 {
		err = fmt.Errorf("unexpected arguments; run `go run ./internal/verify-publication --help`")
	} else {
		err = verify()
	}
	if err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"ok": false, "errors": []string{err.Error() + "; rerun `just verify-publication` after correcting the failure"}})
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "file": publication.RecordPath})
}
