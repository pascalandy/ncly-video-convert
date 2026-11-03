// Package publication binds the bounded publication proof to its current inputs.
package publication

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pascalandy/nicely/sdk/verification"
)

const Directory = "verification/publication"
const RecordPath = Directory + "/record.json"

// Result is one executable publication probe or control. The SDK's separate
// TLC record owns all nine model outcomes.
type Result struct {
	Name string `json:"name"`
	Exit int    `json:"exit"`
}

type Record struct {
	Version     int               `json:"version"`
	ConfirmedAt string            `json:"confirmed_at"`
	BaseCommit  string            `json:"base_commit"`
	SDKVersion  string            `json:"sdk_version"`
	Inputs      map[string]string `json:"inputs"`
	Results     []Result          `json:"results"`
}

type module struct {
	Dir, Version string
	Replace      *module
}

// Inputs hashes source bytes, not timestamps or the caller's checkout path.
// The SDK's resolved source and go.mod also bind a local module replacement.
func Inputs(root string) (map[string]string, string, error) {
	sdk, version, err := resolvedSDK(root)
	if err != nil {
		return nil, "", err
	}
	inputs := map[string]string{"nicely/version": version}
	for _, pattern := range []string{
		"*.go", "cmd/ncly-video-convert/*.go", "manifest.toml", "locales/*.toml", "go.mod", "go.sum",
		Directory + "/*.tla", Directory + "/*.cfg", "verification/tlc.json",
		"internal/publication/*.go", "internal/verify-publication/*.go", "justfile",
	} {
		paths, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil || len(paths) == 0 {
			return nil, "", fmt.Errorf("missing proof inputs for %s: %v", pattern, err)
		}
		for _, path := range paths {
			name, err := filepath.Rel(root, path)
			if err != nil {
				return nil, "", err
			}
			if err := hashFile(inputs, filepath.ToSlash(name), path); err != nil {
				return nil, "", err
			}
		}
	}
	for _, name := range []string{
		"go.mod", "sdk/extension/handle.go", "sdk/runner/signal.go", "sdk/contract/answer.go", "sdk/command/answer.go",
		"sdk/verification/verification.go", "sdk/verification/tlc.go", "sdk/verification/record.go", "sdk/extensionrepo/owned_run.go",
	} {
		if err := hashFile(inputs, "nicely/"+name, filepath.Join(sdk.Dir, name)); err != nil {
			return nil, "", err
		}
	}
	return inputs, version, nil
}

func resolvedSDK(root string) (module, string, error) {
	for attempt := 0; attempt < 2; attempt++ {
		cmd := exec.Command("go", "list", "-m", "-json", "github.com/pascalandy/nicely")
		cmd.Dir, cmd.Env = root, append(os.Environ(), "GOWORK=off", "GOFLAGS=")
		data, err := cmd.Output()
		if err != nil {
			return module{}, "", fmt.Errorf("resolve pinned SDK: %w", err)
		}
		var sdk module
		if err := json.Unmarshal(data, &sdk); err != nil {
			return module{}, "", err
		}
		version := sdk.Version
		if sdk.Replace != nil {
			sdk = *sdk.Replace
		}
		if sdk.Dir != "" {
			return sdk, version, nil
		}
		// A cold module cache can hold metadata without the covered Go files.
		download := exec.Command("go", "mod", "download", "github.com/pascalandy/nicely")
		download.Dir, download.Env = root, cmd.Env
		if output, err := download.CombinedOutput(); err != nil {
			return module{}, "", fmt.Errorf("download pinned SDK: %w: %s", err, output)
		}
	}
	return module{}, "", fmt.Errorf("pinned SDK source unavailable; run `go mod download github.com/pascalandy/nicely`")
}

func hashFile(inputs map[string]string, name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	inputs[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	return nil
}

// CheckRecord runs without Java. Any drift requires the complete proof again.
func CheckRecord(root string) error {
	if problems := verification.CheckRecord(root); len(problems) != 0 {
		return fmt.Errorf("publication models: %s", strings.Join(problems, "; "))
	}
	inputs, version, err := Inputs(root)
	if err != nil {
		return fmt.Errorf("publication inputs: %w; run `just verify-publication`", err)
	}
	data, err := os.ReadFile(filepath.Join(root, RecordPath))
	if err != nil {
		return fmt.Errorf("publication record absent or unreadable: %w; run `just verify-publication`", err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("publication record invalid: %w; run `just verify-publication`", err)
	}
	if record.Version != 2 || record.SDKVersion != version || record.BaseCommit == "" || record.ConfirmedAt == "" {
		return fmt.Errorf("publication record incomplete; run `just verify-publication`")
	}
	if !maps.Equal(record.Inputs, inputs) {
		changed := map[string]bool{}
		for name, hash := range inputs {
			if record.Inputs[name] != hash {
				changed[name] = true
			}
		}
		for name := range record.Inputs {
			if _, present := inputs[name]; !present {
				changed[name] = true
			}
		}
		return fmt.Errorf("publication record stale for %s; run `just verify-publication`", strings.Join(slices.Sorted(maps.Keys(changed)), ", "))
	}
	if len(record.Results) != 3 {
		return fmt.Errorf("publication executable results missing; run `just verify-publication`")
	}
	for i, want := range []Result{{Name: "probe"}, {Name: "mutation", Exit: 1}, {Name: "restored"}} {
		if record.Results[i] != want {
			return fmt.Errorf("publication result %s invalid; run `just verify-publication`", want.Name)
		}
	}
	return nil
}
