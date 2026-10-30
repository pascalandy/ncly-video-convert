# video-convert QA

This file answers one question: what does the UAT of `ncly video convert` add to the [UAT protocol](https://github.com/pascalandy/nicely/blob/main/docs/qa/protocol.md) of Nicely? [spec.md](spec.md) holds the behavior.

## Prerequisites

- `just acceptance` passes on the fixture that [spec.md](spec.md#acceptance) names in `## Acceptance`, in a clone at the tested tag. It checks the Result, the Fidelity, and the Determinism below, and that a still image is refused
- HandBrakeCLI, at the version that [vision.md](vision.md) names as tested
- `ffprobe` and `ffmpeg`, for the independent check. The command runs `ffmpeg` too, to normalize the audio
- On Linux with an Intel GPU, Intel's media runtime, such as `vpl-gpu-rt` on Arch Linux, so that Quick Sync encodes
- The fixture that [spec.md](spec.md#acceptance) names in `## Acceptance`. On a host without it, copy it with the [verified copy procedure](https://github.com/pascalandy/nicely/blob/main/docs/qa/fixtures.md#copy-a-fixture-to-a-host) of the fixture registry, and check it against its `SHA256SUMS`

## Prompt for the fresh agent

"Use Nicely to convert the video `<run>/clip.mov`"

## Criteria

| Family | Criterion |
|---|---|
| Result | `ffmpeg -v error -xerror -i <output> -f null -` exits 0. For the fixture, the output is H.264 1920x1080 with AAC. With `--codec h265`, it is H.265 Main 10 with the `hvc1` tag |
| Encoder | On a machine with a GPU encoder that HandBrake lists, the answer names it: `vt_h264` on a Mac, `qsv_h264` on Linux with Quick Sync. Elsewhere, it names `x264` |
| Fidelity | The output keeps the properties that [spec.md](spec.md#acceptance) names in `## Acceptance` |
| Loudness | `ffmpeg -i <output> -map 0:a:0 -af loudnorm=print_format=json -f null -` measures -16 LUFS within 0.5 LU, and a true peak of at most -1 dBTP. The audio is at 48 kHz |
| Determinism | Two runs on one host give the same frame count and duration, the same audio stream hash (`ffmpeg -v error -i <output> -map 0:a -c copy -f streamhash -hash sha256 -`), and a close video (`ffmpeg -i <first> -i <second> -lavfi ssim -f null -`). With a CPU encoder, the video SSIM is at least 0.999, and the whole-file checksum must match on macOS, since the Linux build of HandBrakeCLI can encode a slightly different video stream. A GPU encoder never gives the same file twice: its video SSIM is at least 0.99, such as 0.995 for two runs of `vt_h264` on the fixture |

## Error cases

| Case | Expected, as a dry run and as a run |
|---|---|
| Fake file, empty file, or truncated video | Exit 2 with `VIDEO_CONVERT_UNREADABLE`, and the message names the file |
| Missing path | Exit 2 with `USAGE_INVALID` |
| Invalid `--timeout`, such as `soon`, `0`, or `-1s` | Exit 2 with `USAGE_INVALID`. A bare number, such as `3`, counts as seconds |
| `--timeout` that ends during the encode, such as `3s` for a 1-minute 1080p clip | The run exits 1 with `RUNTIME`, and the message names the given timeout. The dry run exits 0, unless the scan takes longer than the timeout, such as with `1ms` |
| Existing output, without `--force` | Exit 2 with `CONFIRMATION_REQUIRED` |
| Existing output, with `--force` | Exit 0. The dry run answers `"overwrite":true`, and the run replaces the output |
| Output folder in read-only mode | Exit 1 with `RUNTIME`, and the message says "permission denied" |
| Input that is a symbolic link | Exit 0. The output goes beside the link, with the name of the link |
| Output that is a symbolic link, with or without `--force` | Exit 2 with `USAGE_INVALID` |
| Ctrl-C during the scan of the dry run or during the encode | Exit 130 with `INTERRUPTED` |
| SIGTERM to `ncly` during the scan of the dry run or during the encode | Exit 143 with `TERMINATED` |
| Still image, such as a PNG or a JPEG | Exit 2 with `VIDEO_CONVERT_UNREADABLE`, as [spec.md](spec.md#acceptance) says in `## Acceptance`. The message and hint name the file |

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
