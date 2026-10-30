package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pascalandy/ncly-video-convert/internal/publication"
)

type tools struct {
	Java   string `json:"java"`
	URL    string `json:"tlc_url"`
	SHA256 string `json:"tlc_sha256"`
}

type run struct{ root, scratch, tree string }

func verify() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("the inotify publication probe requires Linux; run `just verify-publication` on Linux")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	inputs, sdkVersion, err := publication.Inputs(root)
	if err != nil {
		return fmt.Errorf("read proof inputs: %w; run `just verify-publication` from the repository root", err)
	}
	scratch, err := os.MkdirTemp("", "ncly-publication-")
	if err != nil {
		return err
	}
	r := run{root: root, scratch: scratch, tree: filepath.Join(scratch, "source")}
	fmt.Fprintln(os.Stderr, "Publication logs:", scratch)
	if err := r.copyTree(); err != nil {
		return err
	}
	snapshot, _, err := publication.Inputs(r.tree)
	if err != nil || !maps.Equal(inputs, snapshot) {
		return fmt.Errorf("source changed during snapshot: %v; run `just verify-publication` again", err)
	}
	pins, err := r.setupTools()
	if err != nil {
		return err
	}
	var results []publication.Result
	for _, check := range publication.Checks {
		result, err := r.model(pins, check.Name, check.Violation)
		if err != nil {
			return err
		}
		results = append(results, result)
	}
	if err := r.probe("probe", "^TestFVPublication", 0, ""); err != nil {
		return err
	}
	results = append(results, publication.Result{Name: "probe"})
	if err := r.mutation(); err != nil {
		return err
	}
	results = append(results, publication.Result{Name: "mutation", Exit: 1}, publication.Result{Name: "restored"})
	current, _, err := publication.Inputs(root)
	if err != nil || !maps.Equal(inputs, current) {
		return fmt.Errorf("source changed during verification: %v; run `just verify-publication` again", err)
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	head, err := cmd.Output()
	if err != nil {
		return err
	}
	record := publication.Record{
		Version: 1, ConfirmedAt: time.Now().UTC().Format(time.RFC3339),
		SourceCommit: strings.TrimSpace(string(head)), SDKVersion: sdkVersion, Inputs: inputs, Results: results,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	// Replace the old record only after every check and restoration succeeded.
	path := filepath.Join(root, publication.RecordPath)
	staged, err := os.CreateTemp(filepath.Dir(path), ".record-*")
	if err != nil {
		return err
	}
	if _, err := staged.Write(append(data, '\n')); err != nil {
		_ = staged.Close()
		return err
	}
	if err := staged.Close(); err != nil {
		return err
	}
	return os.Rename(staged.Name(), path)
}

func (r run) copyTree() error {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	cmd.Dir = r.root
	paths, err := cmd.Output()
	if err != nil {
		return err
	}
	for _, name := range bytes.Split(paths, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		source := filepath.Join(r.root, string(name))
		info, err := os.Lstat(source)
		if os.IsNotExist(err) {
			continue // A tracked deletion belongs to the current tree too.
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("snapshot input %s is not regular; inspect it before `just verify-publication`", name)
		}
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		destination := filepath.Join(r.tree, string(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

// command keeps detailed output off stdout and bounds each external process.
func (r run) command(label string, args ...string) (int, string, error) {
	fmt.Fprintln(os.Stderr, "Publication check:", label)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = r.tree
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "JAVA_TOOL_OPTIONS=-Djava.io.tmpdir="+r.scratch)
	output, err := cmd.CombinedOutput()
	path := filepath.Join(r.scratch, label+".log")
	header, _ := json.Marshal(args)
	if writeErr := os.WriteFile(path, append(append(header, '\n'), output...), 0o600); writeErr != nil {
		return -1, "", writeErr
	}
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && ctx.Err() == nil {
			code = exit.ExitCode()
		} else {
			return -1, string(output), fmt.Errorf("%s: %w; inspect %s and rerun `just verify-publication`", label, err, path)
		}
	}
	return code, string(output), nil
}

func (r run) setupTools() (tools, error) {
	var pins tools
	data, err := os.ReadFile(filepath.Join(r.tree, publication.Directory, "tools.json"))
	if err != nil {
		return pins, err
	}
	if err := json.Unmarshal(data, &pins); err != nil {
		return pins, err
	}
	client := http.Client{Timeout: 90 * time.Second}
	response, err := client.Get(pins.URL)
	if err != nil {
		return pins, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return pins, fmt.Errorf("download TLC: HTTP %d; run `just verify-publication` again", response.StatusCode)
	}
	jar, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return pins, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(jar)) != pins.SHA256 {
		return pins, fmt.Errorf("TLC SHA-256 differs from tools.json; inspect the release before `just verify-publication`")
	}
	if err := os.WriteFile(filepath.Join(r.scratch, "tla2tools.jar"), jar, 0o600); err != nil {
		return pins, err
	}
	// mise installs the exact managed Java version when it is not present.
	code, _, err := r.command("java", "mise", "exec", pins.Java, "--", "java", "-version")
	if err != nil || code != 0 {
		return pins, fmt.Errorf("pinned Java setup failed: exit %d, %v; inspect %s/java.log and rerun `just verify-publication`", code, err, r.scratch)
	}
	return pins, nil
}

var states = regexp.MustCompile(`([\d,]+) distinct states found`)
var depth = regexp.MustCompile(`depth of the complete state graph search is (\d+)`)

func (r run) model(pins tools, name, violation string) (publication.Result, error) {
	result := publication.Result{Name: name, Violation: violation}
	code, output, err := r.command("tlc-"+name,
		"mise", "exec", pins.Java, "--", "java", "-jar", filepath.Join(r.scratch, "tla2tools.jar"),
		"-workers", "2", "-metadir", filepath.Join(r.scratch, "states-"+name),
		"-config", filepath.Join(publication.Directory, name+".cfg"), filepath.Join(publication.Directory, "Publication.tla"))
	if err != nil {
		return result, err
	}
	want, message := 0, "Model checking completed. No error has been found."
	if violation != "" {
		want, message = 12, "Invariant "+violation+" is violated."
	}
	count, levels := states.FindStringSubmatch(output), depth.FindStringSubmatch(output)
	if code != want || !strings.Contains(output, message) || len(count) != 2 || len(levels) != 2 {
		return result, fmt.Errorf("TLC %s: exit %d, want %d with %q; inspect %s/tlc-%s.log and rerun `just verify-publication`", name, code, want, message, r.scratch, name)
	}
	result.Exit = code
	result.Distinct, _ = strconv.Atoi(strings.ReplaceAll(count[1], ",", ""))
	result.Depth, _ = strconv.Atoi(levels[1])
	return result, nil
}

func (r run) probe(label, pattern string, want int, reason string) error {
	code, output, err := r.command(label, "go", "test", "-tags=publication_fv", "-count=1", "-v", "-timeout=2m", "-run", pattern, ".")
	if err != nil {
		return err
	}
	if code != want || !strings.Contains(output, reason) {
		return fmt.Errorf("%s: exit %d, want %d with %q; inspect %s/%s.log and rerun `just verify-publication`", label, code, want, reason, r.scratch, label)
	}
	if want == 0 {
		for _, test := range []string{
			"TestFVPublicationFailure", "TestFVPublicationSignalAfterCommit/interrupt", "TestFVPublicationSignalAfterCommit/terminated",
		} {
			if !strings.Contains(output, "--- PASS: "+test+" (") {
				return fmt.Errorf("%s did not execute %s; inspect %s/%s.log and rerun `just verify-publication`", label, test, r.scratch, label)
			}
		}
	}
	return nil
}

func (r run) mutation() error {
	path := filepath.Join(r.tree, "convert.go")
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	anchor := "\toutcome = j.transcode(ctx, encoded)"
	if bytes.Count(original, []byte(anchor)) != 1 {
		return fmt.Errorf("encode mutation boundary changed; update internal/verify-publication before `just verify-publication`")
	}
	mutant := bytes.Replace(original, []byte(anchor), []byte("\tif err := os.WriteFile(j.output, []byte(\"partial\\n\"), 0o600); err != nil {\n\t\treturn conversionIOError(j.call, err)\n\t}\n"+anchor), 1)
	if err := os.WriteFile(path, mutant, 0o600); err != nil {
		return err
	}
	mutationErr := r.probe("mutation", "^TestFVPublicationFailure$", 1, `clip.converted.mp4: got "partial\n"`)
	// Restore even when the mutation fails for an unexpected reason.
	if err := os.WriteFile(path, original, 0o600); err != nil {
		return err
	}
	if mutationErr != nil {
		return mutationErr
	}
	return r.probe("restored", "^TestFVPublication", 0, "")
}
