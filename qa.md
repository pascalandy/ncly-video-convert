# video-convert QA

This file answers one question: what does the UAT of `ncly video convert` add to the [UAT protocol](https://github.com/pascalandy/nicely/blob/main/docs/qa/protocol.md) of Nicely? [spec.md](spec.md) holds the behavior.

## Prerequisites

- HandBrakeCLI, at the version that [vision.md](vision.md) names as tested
- `ffprobe` and `ffmpeg`, for the independent check
- The fixture that [spec.md](spec.md#acceptance) names in `## Acceptance`

## Prompt for the fresh agent

"Use Nicely to convert the video `<run>/clip.mov`"

## Criteria

| Family | Criterion |
|---|---|
| Result | `ffmpeg -v error -xerror -i <output> -f null -` exits 0. For the fixture, the output is H.264 1920x1080 with AAC |
| Fidelity | The output keeps the properties that [spec.md](spec.md#acceptance) names in `## Acceptance` |
| Determinism | Runs on one host give the same checksum |

## Error cases

| Input | Expected, as a dry run and as a run |
|---|---|
| Fake file, empty file, or truncated video | Exit 2 with `VIDEO_CONVERT_UNREADABLE`, and the message names the file |
| Missing path | Exit 2 with `USAGE_INVALID` |
| Still image | A refusal, as [spec.md](spec.md#acceptance) says in `## Acceptance`. Today it gives a 0.04 s video ([nicely#351](https://github.com/pascalandy/nicely/issues/351)) |

## Known limits

- HDR HLG input gives an 8-bit output tagged HLG. Nobody checked it on a real screen yet.
- With `--json`, the error goes to stderr, as the contract says ([nicely#352](https://github.com/pascalandy/nicely/issues/352)).

## Gotchas

- One conversion of the fixture takes about 1 minute on the macOS test host.
- Damage that the scan of the dry run misses shows only in the run. [spec.md](spec.md#ncly-video-convert) says what the scan decodes.
