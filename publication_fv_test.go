//go:build linux && publication_fv

package video

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// The same HandBrakeCLI shell-stub interface as testdata/script/signals.txtar.
// Extra private files lengthen real RemoveAll, without changing product code.
const fvHandBrake = `#!/bin/sh
while [ "$#" -gt 0 ]; do
  case "$1" in
    --input) input=$2; shift ;;
    --output) output=$2; shift ;;
    --help) exit 0 ;;
    --scan) echo 'JSON Title Set: {"TitleList":[{"Duration":{"Ticks":90000},"FrameRate":{"Num":30,"Den":1},"AudioList":[]}]}'; exit 0 ;;
  esac
  shift
done
if [ "$FV_CASE" = fail ]; then
  echo partial > "$output"
  exit 1
fi
if [ "$FV_CASE" = signal ]; then
  stage=${output%/*}
  i=0
  while [ "$i" -lt 12000 ]; do
    : > "$stage/cleanup-$i"
    i=$((i+1))
  done
fi
echo complete > "$output"
`

func fvBuild(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "ncly-video-convert")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/ncly-video-convert")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, data)
	}
	return bin
}

func fvFixture(t *testing.T, mode string) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		filepath.Join(bin, "HandBrakeCLI"):       fvHandBrake,
		filepath.Join(dir, "clip.mov"):           "original\n",
		filepath.Join(dir, "clip.converted.mp4"): "previous\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	env := append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"),
		"FV_CASE="+mode, "NCLY_JSON=1", "NCLY_NO_INPUT=1", "NCLY_LANG=en",
		"HOME="+dir, "XDG_CONFIG_HOME="+dir, "XDG_DATA_HOME="+dir, "XDG_STATE_HOME="+dir)
	return dir, env
}

func fvCheckFiles(t *testing.T, dir, want string) {
	t.Helper()
	for name, expected := range map[string]string{
		"clip.mov": "original\n", "clip.converted.mp4": want,
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(data) != expected {
			t.Errorf("%s: got %q, %v; want %q", name, data, err, expected)
		}
	}
	stages, err := filepath.Glob(filepath.Join(dir, ".ncly-video-convert-*"))
	if err != nil || len(stages) != 0 {
		t.Errorf("cleanup left %v, error %v", stages, err)
	}
}

// A user-visible failed encode must keep the previous destination and input.
// This is the real-code mutation oracle, not an assertion about source text.
func TestFVPublicationFailure(t *testing.T) {
	bin := fvBuild(t)
	dir, env := fvFixture(t, "fail")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "clip.mov", "--force", "--no-normalize")
	cmd.Dir, cmd.Env = dir, env
	data, err := cmd.CombinedOutput()
	var code int
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	}
	if code != 1 || !bytes.Contains(data, []byte(`"code":"RUNTIME"`)) {
		t.Fatalf("failed encode: exit=%d error=%v output=%s", code, err, data)
	}
	fvCheckFiles(t, dir, "previous\n")
	if t.Failed() {
		return
	}
	t.Logf("failed worker: exit=%d, previous output preserved, input preserved, no stage", code)
}

// inotify reports the actual destination commit before this process signals
// the unmodified extension. No handler/context/OS syscall is mocked.
func TestFVPublicationSignalAfterCommit(t *testing.T) {
	bin := fvBuild(t)
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			dir, env := fvFixture(t, "signal")
			fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
			if err != nil {
				t.Fatal(err)
			}
			defer syscall.Close(fd)
			if _, err := syscall.InotifyAddWatch(fd, dir, syscall.IN_MOVED_TO|syscall.IN_CREATE); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "clip.mov", "--force", "--no-normalize")
			cmd.Dir, cmd.Env = dir, env
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			committed := false
			buffer := make([]byte, 16384)
			for !committed && ctx.Err() == nil {
				n, err := syscall.Read(fd, buffer)
				if err == syscall.EAGAIN || err == syscall.EINTR {
					time.Sleep(time.Millisecond)
					continue
				}
				if err != nil {
					cancel()
					_ = cmd.Wait()
					t.Fatal(err)
				}
				for offset := 0; offset < n; {
					event := (*syscall.InotifyEvent)(unsafe.Pointer(&buffer[offset]))
					end := offset + syscall.SizeofInotifyEvent + int(event.Len)
					name := strings.TrimRight(string(buffer[offset+syscall.SizeofInotifyEvent:end]), "\x00")
					if name == "clip.converted.mp4" && event.Mask&syscall.IN_MOVED_TO != 0 {
						committed = true
					}
					offset = end
				}
			}
			if !committed {
				_ = cmd.Wait()
				t.Fatalf("no commit event: %s", output.Bytes())
			}
			if err := cmd.Process.Signal(sig); err != nil {
				_ = cmd.Wait()
				t.Fatal(err)
			}
			err = cmd.Wait()
			want := 128 + int(sig)
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != want {
				t.Fatalf("after observed rename: want exit %d, got %v, output=%s", want, err, output.Bytes())
			}
			code := "INTERRUPTED"
			if sig == syscall.SIGTERM {
				code = "TERMINATED"
			}
			var answer struct {
				OK        bool   `json:"ok"`
				Output    string `json:"output"`
				SizeBytes int64  `json:"size_bytes"`
				Errors    []struct {
					Code string `json:"code"`
				} `json:"errors"`
			}
			if err := json.Unmarshal(output.Bytes(), &answer); err != nil {
				t.Fatalf("invalid signal answer: %v: %s", err, output.Bytes())
			}
			if answer.OK || len(answer.Errors) == 0 || answer.Errors[0].Code != code {
				t.Fatalf("wrong signal verdict: %s", output.Bytes())
			}
			if answer.Output != filepath.Join(dir, "clip.converted.mp4") || answer.SizeBytes != int64(len("complete\n")) {
				t.Fatalf("signal answer lost published output or size: %s", output.Bytes())
			}
			fvCheckFiles(t, dir, "complete\n")
			t.Logf("observed rename -> sent %s -> exit=%d, code=%s, answer names published output and size, input preserved, no stage", sig, want, code)
		})
	}
}
