package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pascalandy/ncly-video-convert/internal/publication"
	"github.com/pascalandy/nicely/sdk/extensionrepo"
	"github.com/pascalandy/nicely/sdk/platform"
	"github.com/pascalandy/nicely/sdk/verification"
)

type run struct {
	root, scratch, tree string
	env                 []string
}

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
	owned, err := extensionrepo.NewOwnedRun(scratch, extensionrepo.Profile{Set: map[string]string{"TMPDIR": scratch}})
	if err != nil {
		return err
	}
	r := run{root: root, scratch: scratch, tree: filepath.Join(scratch, "source"), env: owned.Env}
	fmt.Fprintln(os.Stderr, "Publication logs:", scratch)
	if err := r.copyTree(); err != nil {
		return err
	}
	snapshot, _, err := publication.Inputs(r.tree)
	if err != nil || !maps.Equal(inputs, snapshot) {
		return fmt.Errorf("source changed during snapshot: %v; run `just verify-publication` again", err)
	}
	// The SDK owns TLC acquisition, digest checking, execution and parsing.
	// verification/tlc.json declares the model configurations.
	_, err = verification.Run(verification.Options{Root: r.tree, Dir: scratch, Env: r.env})
	if err != nil {
		return err
	}
	if err := r.probe("probe", "^TestFVPublication", 0, ""); err != nil {
		return err
	}
	results := []publication.Result{{Name: "probe"}}
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
		Version: 2, ConfirmedAt: time.Now().UTC().Format(time.RFC3339),
		BaseCommit: strings.TrimSpace(string(head)), SDKVersion: sdkVersion, Inputs: inputs, Results: results,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	modelRecord, err := os.ReadFile(filepath.Join(r.tree, verification.RecordFile))
	if err != nil {
		return err
	}
	// Publish both records only after all checks and restoration succeeded.
	if err := platform.WriteFile(filepath.Join(root, verification.RecordFile), modelRecord, 0o644); err != nil {
		return err
	}
	return platform.WriteFile(filepath.Join(root, publication.RecordPath), append(data, '\n'), 0o644)
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
	cmd.Env = r.env
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
