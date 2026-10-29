# video-convert spec

`manifest.toml` holds the command's flags, effects, requirements, and result schemas.

## ncly video convert

Success means HandBrakeCLI exited zero and the published output exists as a nonempty regular file. The answer names its absolute path and size, the codec, the HandBrake encoder, and whether the audio was normalized. `clip.mov` produces `clip.converted.mp4` in the input folder. The command uses `Fast 1080p30` and the MP4 container. `--codec` picks the video codec: `h264`, the default, which plays everywhere, or `h265` in 10 bits, which keeps an HDR input in HDR.

The encoder follows the machine. `HandBrakeCLI --help` lists only the encoders that the machine can run, and the command takes the first one of its codec in this order: Apple VideoToolbox (`vt_h264`, `vt_h265_10bit`), Intel Quick Sync (`qsv_h264`, `qsv_h265_10bit`), then the CPU (`x264`, `x265_10bit`). `x264` keeps the preset as it is. Each other encoder gets its own quality, since the preset's RF 22 means another quality on another encoder's scale: VideoToolbox reads 22 on a scale where 100 is the best. Each GPU quality gave at least the VMAF of `x264` on the fixture of `## Acceptance`. VideoToolbox also decodes on the GPU. When a GPU encode exits with an error, the command runs the CPU encoder of the codec once, and the answer warns, since HandBrake never falls back by itself. On Linux with an Intel GPU, a HandBrake build with Quick Sync offers `qsv` decoding in its help. When it lists no Quick Sync encoder for any codec, Intel's media runtime is missing, such as `vpl-gpu-rt` on Arch Linux, and the answer warns.

By default, ffmpeg normalizes the audio after the encode, with the two passes of its `loudnorm` filter. The first pass measures the encoded audio. The second pass writes a copy that keeps the video stream as it is, and encodes the audio in AAC at 160 kbit/s and 48 kHz, since `loudnorm` works at 192 kHz. The target is -16 LUFS integrated, a range of 12 LU, and a true peak of at most -1 dBTP. The filter aims at -1.5 dBTP, since the AAC encode raises the peak. An input without audio, or with silent audio, keeps HandBrake's audio, and so does `--no-normalize`, which needs no ffmpeg.

In both execution and dry run, the command validates a readable regular input, the destination and its folder's read, write and search access, HandBrakeCLI availability, a video title in the input, the encoder, and ffmpeg when it will normalize. One scan of the input, `HandBrakeCLI --input <file> --scan --json`, finds that title and its audio tracks. It reads one title and decodes up to 10 preview frames, encodes nothing, and writes no file. A dry run starts only this scan, `HandBrakeCLI --help`, and `ffmpeg -version`. It creates no staging folder or output, and reports the planned paths, preset, codec, encoder, and normalization. Decoding of the whole video remains a pending check until execution.

Execution writes in a private staging folder beside the destination. Only a successful, nonempty regular output can be published. Without overwrite consent, an atomic hard link refuses a destination created during conversion. With consent, an atomic rename replaces the previous output after success. A final stat proves the published file exists. `--timeout`, two hours by default, bounds the scan, the encode, and the normalization together, and the message of a stop names that value. All return paths remove staging, after the SDK stops HandBrake or ffmpeg on a timeout or signal. A failed forced conversion preserves the previous output.

The input is opened only for reading. Its folder is resolved through the file system, so `..` after a symlink names the file that the kernel opens, and the output goes beside that file. An output that is a symlink, directory, special file, or the same file as the input is refused even with overwrite consent. The command never edits the existing output in place.

## Effects

The sole user write creates or replaces the converted MP4. No network, paid service, key, reusable partial work, or run record is needed.

## Failures

| Condition | Code |
|---|---|
| Input missing, unreadable, or not regular; timeout invalid; codec other than `h264` or `h265`; destination unsafe | `USAGE_INVALID` |
| The scan exits 2 because HandBrake finds no video title, such as in a text, audio-only, empty, or truncated file. The message and hint name the file | `VIDEO_CONVERT_UNREADABLE`, exit 2 |
| Output exists without overwrite consent, including an arrival during conversion | `CONFIRMATION_REQUIRED` |
| HandBrakeCLI missing, or ffmpeg missing for an input with audio, without `--no-normalize` | `PREREQ_MISSING` |
| The output folder cannot be read, written or searched; the scan exits with another nonzero code, prints no title set, or times out; the CPU encode exits nonzero, times out, or produces no nonempty regular output; ffmpeg exits nonzero, prints no measure, or times out; publication or cleanup fails | `RUNTIME` |
| Ctrl-C or SIGTERM | `INTERRUPTED` or `TERMINATED` |

## Warnings

A warning never changes the exit code.

| Condition | Code |
|---|---|
| The GPU encoder exited with an error, so the command ran the CPU encoder of the codec instead | `VIDEO_CONVERT_GPU_FAILED` |
| Linux with an Intel GPU, whose HandBrake build offers Quick Sync but lists no Quick Sync encoder for any codec | `VIDEO_CONVERT_GPU_RUNTIME_MISSING` |

## Limits

HandBrake must support the built-in preset. Availability is checked without parsing its version because distribution builds can print a source revision instead of a release number. The destination filesystem must support hard links for publication without overwrite consent. Dry run decodes only what the scan reads, and reserves no destination. The scan passes a still image, since HandBrake reads it as a one-frame title. HandBrake also exits 2 for an unknown preset, so the scan carries no preset, and an encode that exits 2 stays `RUNTIME`. A broken HandBrake install that cannot load its built-in presets exits 2 on the scan too, so it reports `VIDEO_CONVERT_UNREADABLE` for any input; the command does not parse HandBrake's text to tell the two apart. Cleanup handles ordinary errors, timeouts, Ctrl-C, and SIGTERM; SIGKILL, power loss, and filesystem refusal can prevent any process from cleaning temporary files. So can a HandBrake that ignores a stop signal for the SDK's whole cleanup time, 15 seconds, since ncly then stops the extension too. Such temporary data is never published as the output. HandBrake inherits ordinary environment variables under the SDK's environment rules. The command discards HandBrake's stdout during the encode, where HandBrake prints encode progress, so the SDK's capture limit does not apply to it. A GPU encoder does not give the same file from run to run. Quick Sync decodes nothing, since its decoding stopped HandBrake with exit 3 on the fixture. On om1, `x265_10bit` was about 2.7 times slower than `x264` on the CPU, so `--codec h265` is slow on a machine without a GPU encoder. When a linear gain would pass the true peak, `loudnorm` compresses the loudness range instead.

## Acceptance

The real input is the fixture `IMG_1642.MOV` of the [fixture registry](https://github.com/pascalandy/nicely/blob/main/docs/qa/fixtures.md) of Nicely. The output keeps these properties of the input:

- The orientation on screen
- The duration, within 0.1 s
- An audio track, at -16 LUFS within 0.5 LU, with a true peak of at most -1 dBTP, at 48 kHz

With `--codec h265`, the output is H.265 Main 10 with the `hvc1` tag, which Apple's players need.

The command must refuse a still image, which HandBrakeCLI accepts. [nicely#351](https://github.com/pascalandy/nicely/issues/351) tracks that refusal.
