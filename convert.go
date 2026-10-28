package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pascalandy/nicely/sdk/command"
	"github.com/pascalandy/nicely/sdk/contract"
	"github.com/pascalandy/nicely/sdk/extension"
	"github.com/pascalandy/nicely/sdk/wrap"
	"golang.org/x/sys/unix"
)

const preset = "Fast 1080p30"

type conversionResult struct {
	Input     string `json:"input"`
	Output    string `json:"output"`
	SizeBytes int64  `json:"size_bytes"`
}

func convert(ctx context.Context, call extension.Call) contract.Outcome {
	started := time.Now()
	timeout := 2 * time.Hour
	if value, given := call.Flags["timeout"]; given {
		parsed, ok := command.Duration(value)
		if !ok {
			return conversionError(call, contract.UsageInvalid, "bad_timeout", "timeout_hint", nil)
		}
		timeout = parsed
	}
	input, err := physical(call.Args[0])
	data := map[string]any{"File": call.Args[0], "Preset": preset}
	if err != nil {
		return conversionError(call, contract.UsageInvalid, "no_file", "input_hint", data)
	}
	info, err := os.Stat(input)
	if err != nil || !info.Mode().IsRegular() {
		return conversionError(call, contract.UsageInvalid, "no_file", "input_hint", data)
	}
	file, err := os.Open(input)
	if err != nil {
		return conversionError(call, contract.UsageInvalid, "no_file", "input_hint", data)
	}
	_ = file.Close()
	output := strings.TrimSuffix(input, filepath.Ext(input)) + ".converted.mp4"
	force := call.Flags["force"] == "true"
	data["File"], data["Output"] = input, output
	if problem := checkDestination(call, info, output, force); problem != nil {
		return contract.Outcome{Errors: []contract.Problem{*problem}}
	}
	// Staging, publication and cleanup work in the output's folder, so a dry
	// run fails there with the code that execution would answer. RemoveAll
	// reads that folder to remove a staging folder that still holds a file.
	if err := unix.Access(filepath.Dir(output), unix.R_OK|unix.W_OK|unix.X_OK); err != nil {
		return conversionIOError(call, err)
	}

	// The scan carries no --preset, since HandBrake also exits 2 for an
	// unknown preset, and the decoder would report that as an unreadable input.
	ready := wrap.Run(ctx, call, wrap.Program{
		Name: "HandBrakeCLI", Args: []string{"--input", input, "--scan"}, Timeout: min(timeout, 10*time.Second),
		DiscardStdout: true,
	}, func(end wrap.Ended) contract.Outcome {
		switch end.Exit {
		case 0:
			return contract.Outcome{}
		case 2:
			return conversionError(call, "VIDEO_CONVERT_UNREADABLE", "unreadable", "unreadable_hint", data)
		}
		return end.Failed()
	})
	if len(ready.Errors) != 0 || ctx.Err() != nil {
		return ready
	}
	if call.Flags["dry-run"] == "true" {
		_, _ = fmt.Fprintln(call.Stdout, call.Printer.T("video-convert.plan", data))
		return contract.Outcome{Data: map[string]any{
			"dry_run": true, "input": input, "output": output, "preset": preset,
			"overwrite": force, "pending_checks": []string{"video_decode"},
		}}
	}
	timeout -= time.Since(started)
	if timeout <= 0 {
		return conversionError(call, contract.Runtime, "no_output", "retry_hint", nil)
	}
	result := encode(ctx, call, input, output, info, force, timeout)
	if value, ok := result.Data.(conversionResult); ok && len(result.Errors) == 0 && ctx.Err() == nil {
		data["Size"] = call.Printer.Size(value.SizeBytes)
		_, _ = fmt.Fprintln(call.Stdout, call.Printer.T("video-convert.result", data))
	}
	return result
}

func encode(ctx context.Context, call extension.Call, input, output string, inputInfo os.FileInfo, force bool, timeout time.Duration) (outcome contract.Outcome) {
	stage, err := os.MkdirTemp(filepath.Dir(output), ".ncly-video-convert-")
	if err != nil {
		return conversionIOError(call, err)
	}
	defer func() {
		if err := os.RemoveAll(stage); err != nil {
			problem := conversionIOError(call, err)
			outcome.Errors = append(outcome.Errors, problem.Errors...)
		}
	}()
	staged := filepath.Join(stage, "output.mp4")
	return wrap.Run(ctx, call, wrap.Program{
		Name: "HandBrakeCLI", Timeout: timeout,
		Args:          []string{"--input", input, "--output", staged, "--preset", preset, "--format", "av_mp4", "--verbose=0"},
		DiscardStdout: true,
	}, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			return end.Failed()
		}
		info, err := os.Lstat(staged)
		if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
			return conversionError(call, contract.Runtime, "no_output", "retry_hint", nil)
		}
		if ctx.Err() != nil {
			return contract.Outcome{}
		}
		if problem := checkDestination(call, inputInfo, output, force); problem != nil {
			return contract.Outcome{Errors: []contract.Problem{*problem}}
		}
		if force {
			err = os.Rename(staged, output)
		} else {
			// Link creates a complete output atomically and refuses an existing
			// destination even if another process creates it after the check.
			err = os.Link(staged, output)
		}
		if os.IsExist(err) {
			return conversionError(call, contract.ConfirmationRequired, "exists", "force_hint", map[string]any{"Output": output})
		}
		if err != nil {
			return conversionIOError(call, err)
		}
		published, err := os.Stat(output)
		if err != nil || !published.Mode().IsRegular() || published.Size() != info.Size() {
			return conversionError(call, contract.Runtime, "no_output", "retry_hint", nil)
		}
		return contract.Outcome{Data: conversionResult{Input: input, Output: output, SizeBytes: published.Size()}}
	})
}

// physical returns the absolute path of the file that the kernel opens for
// name. filepath.Abs, Join and Dir clean ".." as text, which reaches another
// file when ".." follows a symlink, so the folder of name is resolved through
// the file system and its last element is kept as given.
func physical(name string) (string, error) {
	dir, file := filepath.Split(name)
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd + string(filepath.Separator) + dir
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, file), nil
}

func checkDestination(call extension.Call, input os.FileInfo, output string, force bool) *contract.Problem {
	info, err := os.Lstat(output)
	if os.IsNotExist(err) {
		return nil
	}
	var outcome contract.Outcome
	switch {
	case err != nil:
		outcome = conversionIOError(call, err)
	case !info.Mode().IsRegular() || os.SameFile(input, info):
		outcome = conversionError(call, contract.UsageInvalid, "unsafe_output", "output_hint", map[string]any{"Output": output})
	case !force:
		outcome = conversionError(call, contract.ConfirmationRequired, "exists", "force_hint", map[string]any{"Output": output})
	default:
		return nil
	}
	return &outcome.Errors[0]
}

func conversionIOError(call extension.Call, err error) contract.Outcome {
	return conversionError(call, contract.Runtime, "io_error", "output_hint", map[string]any{"Detail": err.Error()})
}

func conversionError(call extension.Call, code contract.Code, message, hint string, data map[string]any) contract.Outcome {
	return contract.Outcome{Errors: []contract.Problem{{
		Code: code, Message: call.Printer.T("video-convert."+message, data), Hint: call.Printer.T("video-convert."+hint, data),
	}}}
}
