package video

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pascalandy/nicely/sdk/command"
	"github.com/pascalandy/nicely/sdk/contract"
	"github.com/pascalandy/nicely/sdk/extension"
	"github.com/pascalandy/nicely/sdk/wrap"
	"golang.org/x/sys/unix"
)

const preset = "Fast 1080p30"

// An encoder is one video encoder of HandBrake and the arguments that follow
// the preset. Each encoder but the preset's own x264 takes its own quality,
// since the preset's RF 22 means another quality on another encoder's scale.
// Each GPU quality gave at least the VMAF of x264 on IMG_1642.MOV.
type encoder struct {
	Name string
	Args []string
}

// codecs holds the encoders of each --codec value in order of preference:
// the GPU ones, then the CPU one, which every HandBrakeCLI build has.
// HandBrakeCLI --help lists only the encoders that the machine can run.
var codecs = map[string][]encoder{
	"h264": {
		{"vt_h264", []string{"--encoder", "vt_h264", "--quality", "65", "--enable-hw-decoding", "videotoolbox"}},
		{"qsv_h264", []string{"--encoder", "qsv_h264", "--quality", "24"}},
		{"x264", nil},
	},
	"h265": {
		{"vt_h265_10bit", []string{"--encoder", "vt_h265_10bit", "--quality", "55", "--enable-hw-decoding", "videotoolbox"}},
		{"qsv_h265_10bit", []string{"--encoder", "qsv_h265_10bit", "--quality", "24"}},
		{"x265_10bit", []string{"--encoder", "x265_10bit", "--quality", "22", "--encoder-preset", "veryfast"}},
	},
}

// loudness is the loudnorm target of the earlier video tools of the author:
// -16 LUFS and a range of 12 LU. The true peak aims at -1.5 dBTP, since the
// AAC encode after loudnorm raised a peak of -1 dBTP to -0.77 dBTP.
const loudness = "loudnorm=I=-16:LRA=12:TP=-1.5"

type conversionResult struct {
	Input      string `json:"input"`
	Output     string `json:"output"`
	SizeBytes  int64  `json:"size_bytes"`
	Codec      string `json:"codec"`
	Encoder    string `json:"encoder"`
	Normalized bool   `json:"normalized"`
}

// A job is one conversion, as preparation planned it.
type job struct {
	call      extension.Call
	input     string
	output    string
	inputInfo os.FileInfo
	force     bool
	codec     string
	encoder   encoder
	normalize bool
	timeout   time.Duration
	deadline  time.Time
	warnings  []contract.Problem
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
	codec := cmp.Or(call.Flags["codec"], "h264")
	input, err := physical(call.Args[0])
	data := map[string]any{"File": call.Args[0], "Preset": preset, "Codec": codec}
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

	j := job{
		call: call, input: input, output: output, inputInfo: info, force: force,
		codec: codec, timeout: timeout, deadline: started.Add(timeout),
	}
	audio, ready := j.scan(ctx, data)
	if len(ready.Errors) != 0 || ctx.Err() != nil {
		return ready
	}
	j.normalize = audio && call.Flags["no-normalize"] != "true"
	if ready = j.prepare(ctx); len(ready.Errors) != 0 || ctx.Err() != nil {
		ready.Warnings = j.warnings
		return ready
	}
	data["Encoder"], data["Normalize"] = j.encoder.Name, j.normalize
	if call.Flags["dry-run"] == "true" {
		_, _ = fmt.Fprintln(call.Stdout, call.Printer.T("video-convert.plan", data))
		return contract.Outcome{Warnings: j.warnings, Data: map[string]any{
			"dry_run": true, "input": input, "output": output, "preset": preset, "codec": codec,
			"encoder": j.encoder.Name, "normalize": j.normalize, "overwrite": force,
			"pending_checks": []string{"video_decode"},
		}}
	}
	result := j.encode(ctx)
	result.Warnings = j.warnings
	if value, ok := result.Data.(conversionResult); ok && len(result.Errors) == 0 && ctx.Err() == nil {
		data["Size"], data["Encoder"], data["Normalize"] = call.Printer.Size(value.SizeBytes), value.Encoder, value.Normalized
		_, _ = fmt.Fprintln(call.Stdout, call.Printer.T("video-convert.result", data))
	}
	return result
}

// prepared bounds each program of preparation, which ends in moments.
const prepared = 10 * time.Second

// scan asks HandBrake for the titles of the input, refuses a still image, and
// reports whether the first title has audio. The scan carries no --preset,
// since HandBrake also exits 2 for an unknown preset, and the decoder would
// report that as an unreadable input.
func (j *job) scan(ctx context.Context, data map[string]any) (bool, contract.Outcome) {
	var audio bool
	call := j.call
	outcome := j.step(ctx, wrap.Program{Name: "HandBrakeCLI", Args: []string{"--input", j.input, "--scan", "--json"}}, prepared, func(end wrap.Ended) contract.Outcome {
		switch end.Exit {
		case 0:
			first, ok := firstTitle(end.Stdout)
			if !ok {
				return end.Malformed()
			}
			if first.still() {
				return conversionError(call, "VIDEO_CONVERT_UNREADABLE", "still_image", "still_image_hint", data)
			}
			audio = len(first.AudioList) != 0
			return contract.Outcome{}
		case 2:
			return conversionError(call, "VIDEO_CONVERT_UNREADABLE", "unreadable", "unreadable_hint", data)
		}
		return end.Failed()
	})
	return audio, outcome
}

// A title is what the scan reports of one title of the input. Ticks count
// 90 kHz, and the frame rate is Num/Den frames a second.
type title struct {
	AudioList []json.RawMessage
	Duration  *struct{ Ticks uint64 }
	FrameRate struct{ Num, Den int64 }
}

// firstTitle reads the title set that HandBrake prints after "JSON Title Set: "
// on stdout, and returns its first title. Its second result is false when
// stdout holds no such title set, or when that title has no duration.
func firstTitle(stdout []byte) (title, bool) {
	_, set, found := bytes.Cut(stdout, []byte("JSON Title Set: "))
	if !found {
		return title{}, false
	}
	var titles struct{ TitleList []title }
	if err := json.NewDecoder(bytes.NewReader(set)).Decode(&titles); err != nil || len(titles.TitleList) == 0 || titles.TitleList[0].Duration == nil {
		return title{}, false
	}
	return titles.TitleList[0], true
}

// still reports whether the title lasts one frame or less, as HandBrake reads
// a still image: 0 ticks for a PNG, and one frame at 25 fps for a JPEG.
// Without a positive frame rate, only a title of 0 ticks is still.
func (t title) still() bool {
	ticks, num, den := t.Duration.Ticks, t.FrameRate.Num, t.FrameRate.Den
	if num <= 0 || den <= 0 {
		return ticks == 0
	}
	// ticks/90000 <= den/num, in 128 bits so that no product overflows.
	hi, lo := bits.Mul64(ticks, uint64(num))
	frameHi, frameLo := bits.Mul64(90000, uint64(den))
	return hi < frameHi || hi == frameHi && lo <= frameLo
}

// prepare picks the encoder, then checks ffmpeg when the job normalizes.
// ffmpeg runs only after the encode, so a missing ffmpeg must fail here,
// where a dry run answers the code that execution would.
func (j *job) prepare(ctx context.Context) contract.Outcome {
	if ready := j.chooseEncoder(ctx); len(ready.Errors) != 0 || ctx.Err() != nil || !j.normalize {
		return ready
	}
	return j.step(ctx, wrap.Program{Name: "ffmpeg", Args: []string{"-hide_banner", "-version"}, DiscardStdout: true}, prepared, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			return end.Failed()
		}
		return contract.Outcome{}
	})
}

// chooseEncoder picks the first encoder of the codec that HandBrakeCLI
// --help lists. On Linux with an Intel GPU, it warns when Quick Sync lacks
// Intel's media runtime.
func (j *job) chooseEncoder(ctx context.Context) contract.Outcome {
	encoders := codecs[j.codec]
	cpu := encoders[len(encoders)-1]
	return j.step(ctx, wrap.Program{Name: "HandBrakeCLI", Args: []string{"--help"}}, prepared, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			return end.Failed()
		}
		help := string(end.Stdout)
		listed := strings.Fields(help)
		j.encoder = cpu
		for _, e := range encoders[:len(encoders)-1] {
			if slices.Contains(listed, e.Name) {
				j.encoder = e
				break
			}
		}
		if runtime.GOOS == "linux" && quickSyncMissing(help) && intelGPU("/sys") {
			j.warn("GPU_RUNTIME_MISSING", "gpu_runtime_missing", "gpu_runtime_hint", nil)
		}
		return contract.Outcome{}
	})
}

// quickSyncMissing reports whether the help of a HandBrake build with Quick
// Sync, which offers 'qsv' decoding, lists no qsv encoder, as when Intel's
// media runtime is missing. A GPU that encodes only some codecs still lists
// their qsv encoders.
func quickSyncMissing(help string) bool {
	return strings.Contains(help, "'qsv'") && !slices.ContainsFunc(strings.Fields(help), func(word string) bool {
		return strings.HasPrefix(word, "qsv_")
	})
}

// intelGPU reports whether a DRM device under sys, such as /sys, comes from
// Intel, whose PCI vendor ID is 0x8086.
func intelGPU(sys string) bool {
	vendors, _ := filepath.Glob(filepath.Join(sys, "class", "drm", "*", "device", "vendor"))
	for _, vendor := range vendors {
		id, err := os.ReadFile(vendor)
		if err == nil && strings.TrimSpace(string(id)) == "0x8086" {
			return true
		}
	}
	return false
}

func (j *job) encode(ctx context.Context) (outcome contract.Outcome) {
	stage, err := os.MkdirTemp(filepath.Dir(j.output), ".ncly-video-convert-")
	if err != nil {
		return conversionIOError(j.call, err)
	}
	defer func() {
		if err := os.RemoveAll(stage); err != nil {
			problem := conversionIOError(j.call, err)
			outcome.Errors = append(outcome.Errors, problem.Errors...)
		}
	}()
	encoded := filepath.Join(stage, "encoded.mp4")
	outcome = j.transcode(ctx, encoded)
	if len(outcome.Errors) != 0 || ctx.Err() != nil {
		return outcome
	}
	final, normalized := encoded, false
	if j.normalize {
		normalizedFile := filepath.Join(stage, "normalized.mp4")
		if normalized, outcome = j.normalizeAudio(ctx, encoded, normalizedFile); len(outcome.Errors) != 0 || ctx.Err() != nil {
			return outcome
		}
		if normalized {
			final = normalizedFile
		}
	}
	return j.publish(final, normalized)
}

// transcode runs HandBrake with the planned encoder. When a GPU encoder exits
// with an error, it runs the CPU encoder of the codec once, with a warning,
// since HandBrake never falls back by itself.
func (j *job) transcode(ctx context.Context, staged string) contract.Outcome {
	encoders := codecs[j.codec]
	cpu := encoders[len(encoders)-1]
	outcome, exited := j.handBrake(ctx, staged)
	if exited && j.encoder.Name != cpu.Name && ctx.Err() == nil {
		j.warn("GPU_FAILED", "gpu_failed", "gpu_failed_hint", map[string]any{"Encoder": j.encoder.Name, "CPU": cpu.Name})
		j.encoder = cpu
		outcome, _ = j.handBrake(ctx, staged)
	}
	return outcome
}

// handBrake runs one encode into staged. Its second result is true when
// HandBrake exited with an error, as opposed to a timeout or a signal.
func (j *job) handBrake(ctx context.Context, staged string) (contract.Outcome, bool) {
	var exited bool
	args := append([]string{"--input", j.input, "--output", staged, "--preset", preset, "--format", "av_mp4", "--verbose=0"}, j.encoder.Args...)
	outcome := j.step(ctx, wrap.Program{Name: "HandBrakeCLI", Args: args, DiscardStdout: true}, 0, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			exited = true
			return end.Failed()
		}
		if info, err := os.Lstat(staged); err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
			return conversionError(j.call, contract.Runtime, "no_output", "retry_hint", nil)
		}
		return contract.Outcome{}
	})
	return outcome, exited
}

// normalizeAudio measures the loudness of the encoded audio with loudnorm,
// then writes a copy of encoded whose audio meets the target, as in
// loudnorm's two-pass mode. loudnorm works at 192 kHz, so the copy returns to
// 48 kHz. Silent audio has no loudness to correct, so it reports false and
// writes nothing.
func (j *job) normalizeAudio(ctx context.Context, encoded, normalized string) (bool, contract.Outcome) {
	var measure string
	outcome := j.step(ctx, wrap.Program{
		Name:          "ffmpeg",
		Args:          []string{"-nostdin", "-hide_banner", "-nostats", "-i", encoded, "-map", "0:a:0", "-af", loudness + ":print_format=json", "-f", "null", "-"},
		DiscardStdout: true,
	}, 0, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			return end.Failed()
		}
		var ok bool
		if measure, ok = measured(end.Stderr); !ok {
			return end.Malformed()
		}
		return contract.Outcome{}
	})
	if len(outcome.Errors) != 0 || ctx.Err() != nil || measure == "" {
		return false, outcome
	}
	outcome = j.step(ctx, wrap.Program{
		Name: "ffmpeg",
		Args: []string{
			"-nostdin", "-hide_banner", "-nostats", "-loglevel", "error", "-i", encoded, "-map", "0:v", "-map", "0:a",
			"-c", "copy", "-c:a", "aac", "-b:a", "160k", "-ar", "48000",
			"-af", loudness + ":" + measure + ":linear=true:print_format=none", normalized,
		},
		DiscardStdout: true,
	}, 0, func(end wrap.Ended) contract.Outcome {
		if end.Exit != 0 {
			return end.Failed()
		}
		if info, err := os.Lstat(normalized); err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
			return conversionError(j.call, contract.Runtime, "no_output", "retry_hint", nil)
		}
		return contract.Outcome{}
	})
	return len(outcome.Errors) == 0, outcome
}

// measured reads the JSON that loudnorm prints last on stderr, and returns its
// measures as the options of the second pass. It returns "" and true for
// silent audio, whose loudness is -inf, and false when stderr holds no
// readable measure.
func measured(stderr []byte) (string, bool) {
	key := bytes.LastIndex(stderr, []byte(`"input_i"`))
	if key < 0 {
		return "", false
	}
	start := bytes.LastIndexByte(stderr[:key], '{')
	if start < 0 {
		return "", false
	}
	var m struct {
		I      string `json:"input_i"`
		TP     string `json:"input_tp"`
		LRA    string `json:"input_lra"`
		Thresh string `json:"input_thresh"`
		Offset string `json:"target_offset"`
	}
	if err := json.NewDecoder(bytes.NewReader(stderr[start:])).Decode(&m); err != nil {
		return "", false
	}
	values := []string{m.I, m.TP, m.LRA, m.Thresh, m.Offset}
	for _, value := range values {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", false
		}
		if math.IsInf(number, 0) {
			return "", true
		}
	}
	return fmt.Sprintf("measured_I=%s:measured_TP=%s:measured_LRA=%s:measured_thresh=%s:offset=%s", values[0], values[1], values[2], values[3], values[4]), true
}

// step runs one program under what remains of --timeout, and under limit
// when it is shorter. wrap names that remainder when the deadline stops the
// program, so the message names --timeout instead, as it does for a program
// that the deadline leaves no time to start.
func (j *job) step(ctx context.Context, p wrap.Program, limit time.Duration, decode func(wrap.Ended) contract.Outcome) contract.Outcome {
	printer := j.call.Printer
	stopped := map[string]any{"Name": p.Name, "Timeout": printer.Duration(j.timeout)}
	rest := time.Until(j.deadline)
	if rest <= 0 {
		return contract.Outcome{Errors: []contract.Problem{{Code: contract.Runtime, Message: printer.T("wrap.timeout", stopped), Hint: printer.T("outcome.check_hint")}}}
	}
	p.Timeout = rest
	if limit > 0 && limit < rest {
		p.Timeout = limit
	}
	outcome := wrap.Run(ctx, j.call, p, decode)
	remainder := map[string]any{"Name": p.Name, "Timeout": printer.Duration(rest)}
	if p.Timeout == rest && len(outcome.Errors) != 0 && outcome.Errors[0].Message == printer.T("wrap.timeout", remainder) {
		outcome.Errors[0].Message = printer.T("wrap.timeout", stopped)
	}
	return outcome
}

// publish moves the complete output from staging to its destination.
func (j *job) publish(staged string, normalized bool) contract.Outcome {
	info, err := os.Lstat(staged)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return conversionError(j.call, contract.Runtime, "no_output", "retry_hint", nil)
	}
	if problem := checkDestination(j.call, j.inputInfo, j.output, j.force); problem != nil {
		return contract.Outcome{Errors: []contract.Problem{*problem}}
	}
	if j.force {
		err = os.Rename(staged, j.output)
	} else {
		// Link creates a complete output atomically and refuses an existing
		// destination even if another process creates it after the check.
		err = os.Link(staged, j.output)
	}
	if os.IsExist(err) {
		return conversionError(j.call, contract.ConfirmationRequired, "exists", "force_hint", map[string]any{"Output": j.output})
	}
	if err != nil {
		return conversionIOError(j.call, err)
	}
	published, err := os.Stat(j.output)
	if err != nil || !published.Mode().IsRegular() || published.Size() != info.Size() {
		return conversionError(j.call, contract.Runtime, "no_output", "retry_hint", nil)
	}
	return contract.Outcome{Data: conversionResult{
		Input: j.input, Output: j.output, SizeBytes: published.Size(),
		Codec: j.codec, Encoder: j.encoder.Name, Normalized: normalized,
	}}
}

// warn adds a warning with the extension's prefix, which needs no entry in
// the manifest.
func (j *job) warn(code, message, hint string, data map[string]any) {
	j.warnings = append(j.warnings, contract.Problem{
		Code:    contract.Code("VIDEO_CONVERT_" + code),
		Message: j.call.Printer.T("video-convert."+message, data),
		Hint:    j.call.Printer.T("video-convert."+hint, data),
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
