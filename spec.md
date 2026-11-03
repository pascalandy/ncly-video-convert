# video-convert spec

`manifest.toml` holds the command's flags, effects, requirements, and result schemas.

Each line of Failures and Acceptance is `proven` by a test claim. Untested lines stay under `## Open lines` until their scenarios pass.

## ncly video convert

Success means HandBrakeCLI exited zero and the published output exists as a nonempty regular file. The answer names its absolute path and size, the codec, the HandBrake encoder, and whether the audio was normalized. `clip.mov` produces `clip.converted.mp4` in the input folder. The command uses `Fast 1080p30` and the MP4 container. `--codec` picks the video codec: `h264`, the default, which plays everywhere, or `h265` in 10 bits, which keeps an HDR input in HDR.

The encoder follows the machine. `HandBrakeCLI --help` lists only the encoders that the machine can run, and the command takes the first one of its codec in this order: Apple VideoToolbox (`vt_h264`, `vt_h265_10bit`), Intel Quick Sync (`qsv_h264`, `qsv_h265_10bit`), then the CPU (`x264`, `x265_10bit`). `x264` keeps the preset as it is. Each other encoder gets its own quality, since the preset's RF 22 means another quality on another encoder's scale: VideoToolbox reads 22 on a scale where 100 is the best. Each GPU quality gave at least the VMAF of `x264` on the fixture of `## Acceptance`. VideoToolbox also decodes on the GPU. When a GPU encode exits with an error, the command runs the CPU encoder of the codec once, and the answer warns, since HandBrake never falls back by itself. On Linux with an Intel GPU, a HandBrake build with Quick Sync offers `qsv` decoding in its help. When it lists no Quick Sync encoder for any codec, Intel's media runtime is missing, such as `vpl-gpu-rt` on Arch Linux, and the answer warns.

By default, ffmpeg normalizes the audio after the encode, with the two passes of its `loudnorm` filter. The first pass measures the encoded audio. The second pass writes a copy that keeps the video stream as it is, and encodes the audio in AAC at 160 kbit/s and 48 kHz, since `loudnorm` works at 192 kHz. The target is -16 LUFS integrated, a range of 12 LU, and a true peak of at most -1 dBTP. The filter aims at -1.5 dBTP, since the AAC encode raises the peak. An input without audio, or with silent audio, keeps HandBrake's audio, and so does `--no-normalize`, which needs no ffmpeg.

In both execution and dry run, the command validates a readable regular input, the destination and its folder's read, write and search access, HandBrakeCLI availability, a video title of more than one frame in the input, the encoder, and ffmpeg when it will normalize. One scan of the input, `HandBrakeCLI --input <file> --scan --json`, finds that title, its duration, and its audio tracks. It reads one title and decodes up to 10 preview frames, encodes nothing, and writes no file. HandBrake reads a still image as a title of one frame or less, such as 0 ticks of its 90 kHz clock for a PNG, and one frame at 25 fps for a JPEG, so the scan refuses such a title. A dry run starts only this scan, `HandBrakeCLI --help`, and `ffmpeg -version`. It creates no staging folder or output, and reports the planned paths, preset, codec, encoder, and normalization. Decoding of the whole video remains a pending check until execution. The dry run also reports the publication operation as pending, because checking access cannot prove filesystem support for a hard link or rename.

Execution writes in a private staging folder beside the destination. Only a successful, nonempty regular output can be published. Without overwrite consent, an atomic hard link refuses a destination created during conversion. With consent, an atomic rename replaces the previous output after success. A final stat proves the published file exists. `--timeout`, two hours by default, bounds the scan, the encode, and the normalization together, and the message of a stop names that value. Every ordinary return attempts staging cleanup, after the SDK stops HandBrake or ffmpeg on a timeout or signal. Successful cleanup leaves no stage; a filesystem cleanup refusal is reported as RUNTIME. The runtime constraints below bound this guarantee. A failed forced conversion preserves the previous output.

The input is opened only for reading. Its folder is resolved through the file system, so `..` after a symlink names the file that the kernel opens, and the output goes beside that file. An output that is a symlink, directory, special file, or the same file as the input is refused even with overwrite consent. The command never edits the existing output in place. HandBrake keeps its existing metadata behavior. Display declarations remain effective in the decoded video. The published file keeps the completed staged file's mode, as selected by HandBrake or ffmpeg under the caller's umask. Force mode replaces the old file and its mode together.

**Limits.**

| ID | State | Limit | Accepted |
|---|---|---|---|
| `private-metadata` | limit | The output does not scrub descriptive metadata. Author, capture date and GPS location can remain. This is the accepted privacy loss for the Output cases that keep private data and keep data a reader needs. Opt-in stripping is tracked in [#18](https://github.com/pascalandy/ncly-video-convert/issues/18) | Head Chef decision relayed by Chef 2 on 2026-11-03 in brigade a1d9, recorded in [PR #17](https://github.com/pascalandy/ncly-video-convert/pull/17) |
| `kit-accepted-limits` | limit | SDK v0.26.0 still demands privacy-removal reads for an accepted retention limit. [nicely#462](https://github.com/pascalandy/nicely/issues/462) must ship in Nicely before this build can pass check and signoff. Coverage remains failed; no waiver is taken | Head Chef D068 decision relayed by Chef 2 on 2026-11-03, recorded in [PR #17](https://github.com/pascalandy/ncly-video-convert/pull/17) |

## Effects

The sole user write creates or replaces the converted MP4. No network, paid service, key, reusable partial work, or run record is needed.

## Failures

| ID | Condition | Code |
|---|---|---|
| `bad-input` | Input missing, unreadable, or not regular | `USAGE_INVALID` |
| `bad-timeout` | Timeout invalid | `USAGE_INVALID` |
| `bad-codec` | Codec other than `h264` or `h265` | `USAGE_INVALID` |
| `unsafe-output` | Destination unsafe | `USAGE_INVALID` |
| `no-video-title` | The scan exits 2 because HandBrake finds no video title, such as in a text, audio-only, empty, or truncated file. The message and hint name the file | `VIDEO_CONVERT_UNREADABLE`, exit 2 |
| `output-exists` | Output exists without overwrite consent | `CONFIRMATION_REQUIRED` |
| `output-arrives` | Output arrives during conversion, without overwrite consent | `CONFIRMATION_REQUIRED` |
| `no-handbrake` | HandBrakeCLI missing | `PREREQ_MISSING` |
| `no-ffmpeg` | ffmpeg missing for an input with audio, without `--no-normalize` | `PREREQ_MISSING` |
| `folder-access` | The output folder cannot be read, written or searched | `RUNTIME` |
| `scan-failed` | The scan exits with another nonzero code, prints no title set or no duration of the first title, or times out | `RUNTIME` |
| `encode-failed` | The CPU encode exits nonzero, times out, or produces no nonempty regular output | `RUNTIME` |
| `normalize-failed` | ffmpeg exits nonzero, prints no measure, or times out | `RUNTIME` |
| `publish-failed` | Publication or cleanup fails | `RUNTIME` |
| `interrupted` | Ctrl-C | `INTERRUPTED` |
| `terminated` | SIGTERM | `TERMINATED` |

## Warnings

A warning never changes the exit code.

| Condition | Code |
|---|---|
| The GPU encoder exited with an error, so the command ran the CPU encoder of the codec instead | `VIDEO_CONVERT_GPU_FAILED` |
| Linux with an Intel GPU, whose HandBrake build offers Quick Sync but lists no Quick Sync encoder for any codec | `VIDEO_CONVERT_GPU_RUNTIME_MISSING` |

## Runtime constraints

HandBrake 1.7.2 through 1.9.x write conversion time as `creation_time`; from 1.10 it copies a source timestamp when one exists. This version-dependent container metadata does not define media determinism. No new HandBrake minimum is required.

HandBrake must support the built-in preset. Availability is checked without parsing its version because distribution builds can print a source revision instead of a release number. The destination filesystem must support hard links for publication without overwrite consent. Dry run decodes only what the scan reads, and reserves no destination. HandBrake also exits 2 for an unknown preset, so the scan carries no preset, and an encode that exits 2 stays `RUNTIME`. A broken HandBrake install that cannot load its built-in presets exits 2 on the scan too, so it reports `VIDEO_CONVERT_UNREADABLE` for any input; the command does not parse HandBrake's text to tell the two apart. Cleanup handles ordinary errors, timeouts, Ctrl-C, and SIGTERM; SIGKILL, power loss, and filesystem refusal can prevent any process from cleaning temporary files. A scenario also checks a worker that ignores SIGTERM: the SDK force-stops it and ordinary cleanup removes staging. An external SIGKILL during cleanup can still interrupt that cleanup. Such temporary data is never published as the output. HandBrake inherits ordinary environment variables under the SDK's environment rules. The command discards HandBrake's stdout during the encode, where HandBrake prints encode progress, so the SDK's capture limit does not apply to it. A GPU encoder does not give the same file from run to run. Quick Sync decodes nothing, since its decoding stopped HandBrake with exit 3 on the fixture. On om1, `x265_10bit` was about 2.7 times slower than `x264` on the CPU, so `--codec h265` is slow on a machine without a GPU encoder. When a linear gain would pass the true peak, `loudnorm` compresses the loudness range instead.

## Acceptance

The real input is the fixture `IMG_1642.MOV` of the [fixture registry](https://github.com/pascalandy/nicely/blob/main/docs/qa/fixtures.md) of Nicely.

- `orientation`: The output keeps the orientation of the input on screen
- `duration`: The output keeps the duration of the input, within 0.1 s
- `loudness`: The output keeps an audio track, at -16 LUFS within 0.5 LU, with a true peak of at most -1 dBTP, at 48 kHz
- `hevc-output`: With `--codec h265`, the output is H.265 Main 10 with the `hvc1` tag, which Apple's players need

- `old-handbrake`: A CLI without metadata-stripping support still converts successfully under the unchanged default policy
- `publication-cases`: Existing-output, input-link and temporary-path cases construct their preconditions and independently check preserved or published content
- `determinism`: Without normalization, repeated conversions keep frame count, duration and audio packet hashes; video SSIM is at least 0.999 for CPU encoders and 0.99 for GPU encoders, independently of container timestamps
- `cleanup-limits`: Ordinary success, failure and handled signals leave no staging when the filesystem cooperates; the existing runtime constraints bound cleanup
- `publication-pending`: A dry run lists `publication.link` without force and `publication.rename` with force in `pending_checks`
- `metadata-retained`: The acceptance fixtures demonstrate possible retention of descriptive metadata, without promising that every input tag survives

The domain refuses these inputs:

| ID | Input that HandBrakeCLI accepts | Code |
|---|---|---|
| `still-image` | A still image, such as a PNG or a JPEG, which the scan reads as a first title of one frame or less. The message and hint name the file | `VIDEO_CONVERT_UNREADABLE`, exit 2 |

## Case matrix

Each family names the criteria that its accepted and refused inputs prove. Real media checks of orientation, duration, audio and HEVC stay in `acceptance.sh`; the made-media scenario also checks the primary output through an independent reader.

| Family | Source | Accepted | Refused |
|---|---|---|---|
| Readable local regular input | Input: Unreadable | `duration` | `bad-input` |
| Complete video title | Input: Empty; Input: Truncated; Input: Of the wrong type | `duration` | `no-video-title`, `still-image`, `scan-failed` |
| Codec and encoder | Intake: H.264, H.265, GPU selection and CPU fallback | `hevc-output`, `determinism` | `bad-codec`, `encode-failed` |
| Audio normalization | Intake: no audio, silence, measured loudness and AAC | `loudness` | `normalize-failed`, `no-ffmpeg` |
| Tools installed | Other: Missing program; pre-1.10 CLI compatibility | `duration`, `old-handbrake` | `no-handbrake`, `no-ffmpeg` |
| Destination access | Input and output folder access | `duration` | `folder-access`, `unsafe-output` |
| Existing destination and consent | Output: Exists, without --force | `duration` | `output-exists` |
| Output cannot alias the input | Output: Is a link to the input | `duration` | `unsafe-output` |
| Concurrent creation | Output: Appears during the run, without --force | `duration` | `output-arrives` |
| Complete private staging | Output: Temporary file | `duration`, `publication-cases` | `encode-failed`, `publish-failed` |
| Publication capability is pending | Other: Dry run | `publication-pending` | `publish-failed` |
| Work has one deadline | Input: Very large; Input: Large expansion; Interruption: Timeout | `duration` | `bad-timeout`, `scan-failed`, `encode-failed` |
| Stop and cleanup | Interruption: SIGINT or SIGTERM during the work | `duration`, `cleanup-limits` | `interrupted`, `terminated` |
| Paths stay one argument | File name: Spaces; File name: Leading dash; File name: Shell characters | `duration` | `bad-input` |
| Display orientation | Output: Loses data that changes the meaning | `orientation` | `still-image` |
| Display declarations | Output: Keeps data that a reader needs | `orientation`, `metadata-retained` | None |
| Descriptive metadata retained by policy | Output: Keeps private data of the input | `metadata-retained`, `private-metadata` | None |

Regular-file input excludes DVD directories and executable scripts. The bounded job is the first video title and the preset's selected audio, with the codec and display declarations above; arbitrary arguments and active content are never relayed. Worker output and expensive decode share the declared timeout. The existing operational constraints above still apply.

## Publication

| Command | Mode | Destination | Overwrite | Protected | Operation | Dry run |
|---|---|---|---|---|---|---|
| `ncly video convert` | create | `output` | `overwrite` | `file` | `link` | pending |
| `ncly video convert` | replace | `output` | `overwrite` | `file` | `rename` | pending |

Every obligation is exercised by `predicts`, `appears`, and Linux `without` checks. There are no untested publication obligations.

## Open lines

None. The generic cases above are covered by the declared families and the scenarios of this build.
