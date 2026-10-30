# video-convert QA

This file answers one question: what does the UAT of `ncly video convert` add to the [UAT protocol](https://github.com/pascalandy/nicely/blob/main/docs/qa/protocol.md) of Nicely? [spec.md](spec.md) holds the behavior.

## Prerequisites

- `just acceptance` passes on the fixture that [spec.md](spec.md#acceptance) names in `## Acceptance`, in a clone at the tested tag. It checks the Result, the Fidelity, and the Determinism below, and the refusal of a still image, the `still-image` line of [spec.md](spec.md#acceptance)
- HandBrakeCLI, at the version that [vision.md](vision.md) names as tested
- `ffprobe` and `ffmpeg`, for the independent check. The command runs `ffmpeg` too, to normalize the audio
- On Linux with an Intel GPU, Intel's media runtime, such as `vpl-gpu-rt` on Arch Linux, so that Quick Sync encodes
- The fixture that [spec.md](spec.md#acceptance) names in `## Acceptance`. On a host without it, copy it with the [verified copy procedure](https://github.com/pascalandy/nicely/blob/main/docs/qa/fixtures.md#copy-a-fixture-to-a-host) of the fixture registry, and check it against its `SHA256SUMS`

## Prompt for the fresh agent

"Use Nicely to convert the video `<run>/clip.mov`"

## Criteria

`Covered by` names the test that proves the criterion, or says why no test can.

| Family | Criterion | Covered by |
|---|---|---|
| Result | `ffmpeg -v error -xerror -i <output> -f null -` exits 0. For the fixture, the output is H.264 1920x1080 with AAC. With `--codec h265`, it is H.265 Main 10 with the `hvc1` tag | `just acceptance` |
| Encoder | On a machine with a GPU encoder that HandBrake lists, the answer names it: `vt_h264` on a Mac, `qsv_h264` on Linux with Quick Sync. Elsewhere, it names `x264` | `encoders.txtar` proves the choice from a stub help. Manual: the encoders that HandBrake lists depend on the GPU and the drivers of the host, which no test knows. Read the encoder in the last line of `just acceptance` |
| Fidelity | The output keeps the properties `orientation`, `duration`, and `loudness` of [spec.md](spec.md#acceptance) | `just acceptance` |
| Loudness | `ffmpeg -i <output> -map 0:a:0 -af loudnorm=print_format=json -f null -` measures -16 LUFS within 0.5 LU, and a true peak of at most -1 dBTP. The audio is at 48 kHz | `just acceptance` |
| Determinism | Two runs on one host give the same frame count and duration, the same audio stream hash (`ffmpeg -v error -i <output> -map 0:a -c copy -f streamhash -hash sha256 -`), and a close video (`ffmpeg -i <first> -i <second> -lavfi ssim -f null -`). With a CPU encoder, the video SSIM is at least 0.999, and the whole-file checksum must match on macOS, since the Linux build of HandBrakeCLI can encode a slightly different video stream. A GPU encoder never gives the same file twice: its video SSIM is at least 0.99, such as 0.995 for two runs of `vt_h264` on the fixture | `just acceptance` |

## Error cases

These scenarios run in every `just check`.
The `spec.md` column names the criterion that holds the expected code.
The Expected column adds the other assertions.

| Case | spec.md | Expected, as a dry run and as a run | Scenarios |
|---|---|---|---|
| Fake file, empty file, or truncated video | `no-video-title` | The message and hint name the file | [Fake](testdata/script/convert.txtar), [empty](testdata/script/qa_empty.txtar), [truncated](testdata/script/qa_truncated.txtar) |
| Missing path | `bad-input` | Exit 2 | [convert](testdata/script/convert.txtar) |
| Invalid `--timeout`, such as `soon`, `0`, or `-1s` | `bad-timeout` | Exit 2. A bare number, such as `3`, counts as seconds | [Invalid text](testdata/script/convert.txtar), [numeric values](testdata/script/qa_timeout.txtar) |
| Timeout during the encode | `encode-failed` | The run exits 1 and names the given timeout. The dry run succeeds unless preparation exceeds the timeout | [Scan and encode](testdata/script/convert.txtar), [bare seconds](testdata/script/qa_timeout.txtar) |
| Existing output, without `--force` | `output-exists` | Exit 2 | [convert](testdata/script/convert.txtar) |
| Existing output, with `--force` | None | Exit 0. The dry run answers `"overwrite":true`. The run replaces the output | [convert](testdata/script/convert.txtar) |
| Output folder in read-only mode | `folder-access` | Exit 1. The message says "permission denied" | [convert](testdata/script/convert.txtar) |
| Input that is a symbolic link | None | Exit 0. The output goes beside the link, with the name of the link | [convert](testdata/script/convert.txtar) |
| Output that is a symbolic link, with or without `--force` | `unsafe-output` | Exit 2. The link and its target stay the same | [convert](testdata/script/convert.txtar) |
| Ctrl-C during the preview scan or the encode | `interrupted` | Exit 130 | [signals](testdata/script/signals.txtar) |
| SIGTERM to `ncly` during the preview scan or the encode | `terminated` | Exit 143 | [signals](testdata/script/signals.txtar) |
| Still image, such as a PNG or a JPEG | `still-image` | The message and hint name the file | [still](testdata/script/still.txtar) |

After each of these cases, the input, any previous output, and the target of a link stay the same. No `.ncly-video-convert-*` staging folder and no HandBrakeCLI process stay.

## Uninstall

After the UAT, run `just uninstall` in the checkout of the extension. For an install without that recipe, move these four entries of Nicely's extensions folder to the trash: `ncly-video-convert`, `ncly-video-convert.toml`, `ncly-video-convert.locales/`, and `ncly-video-convert.skill/`. That folder is `nicely/extensions/` in `XDG_DATA_HOME`, or in `~/.local/share` when `XDG_DATA_HOME` is not set. Then `ncly describe video convert --json` must exit 2 with `NOT_FOUND`.

## Known limits

- HDR HLG input gives an 8-bit output tagged HLG with the default H.264. Nobody checked it on a real screen yet. With `--codec h265`, the output stays 10-bit HLG.
- With `--json`, the error goes to stderr, as the contract says ([nicely#352](https://github.com/pascalandy/nicely/issues/352)).
- `locales/` holds only `en.toml`, so under `--lang fr-CA` the help and the errors print in English without a warning, as the contract's fallback to `en` says. Nicely's own text is in English too.

## Gotchas

- One conversion of the fixture takes about 25 s with VideoToolbox on the macOS test host, and about 1 minute with x264, plus about 10 s to normalize the audio. These times grow when the host runs other work.
- Damage that the scan of the dry run misses shows only in the run. [spec.md](spec.md#ncly-video-convert) says what the scan decodes.
- Isolated XDG folders, such as for a scratch install, can stop the `just` that mise installed: with another `XDG_CONFIG_HOME`, its shim finds no version and fails with `No version is set for shim: just`. Run `mise which just` before you change the XDG variables, and call that path.
- A clip without a `creation_time` tag, such as a clip that `ffmpeg` generates, gets the clock's time in its output, so two of its runs never give the same file checksum. HandBrake copies the tag of a source that has one.
