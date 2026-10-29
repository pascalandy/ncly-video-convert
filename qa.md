# video-convert QA

This file answers one question: what does the UAT of `ncly video convert` add to the [UAT protocol](https://github.com/pascalandy/nicely/blob/main/docs/qa/protocol.md) of Nicely? [spec.md](spec.md) holds the behavior.

## Prerequisites

- HandBrakeCLI, tested with 1.11.2, and `ffprobe` and `ffmpeg` for the independent check
- The fixture `IMG_1642.MOV` from the [fixture registry](https://github.com/pascalandy/nicely/blob/main/docs/qa/fixtures.md)

## Prompt for the fresh agent

"Use Nicely to convert the video `<run>/clip.mov`"

## Criteria

| Family | Criterion |
|---|---|
| Result | `ffmpeg -v error -xerror -i <output> -f null -` exits 0. For the fixture, the output is H.264 1920x1080 with AAC |
| Fidelity | The orientation on screen is unchanged. The duration matches within 0.1 s. An audio track is present |
| Determinism | Runs on one host give the same checksum |

## Error cases

| Input | Expected, as a dry run and as a run |
|---|---|
| Fake file, empty file, or truncated video | Exit 2 with `VIDEO_CONVERT_UNREADABLE`, and the message names the file |
| Missing path | Exit 2 with `USAGE_INVALID` |
| Still image | A refusal, once [nicely#351](https://github.com/pascalandy/nicely/issues/351) is fixed. Today it gives a 0.04 s video |

## Known limits

- HDR HLG input gives an 8-bit output tagged HLG. Nobody checked it on a real screen yet.
- With `--json`, the error goes to stderr, as the contract says ([nicely#352](https://github.com/pascalandy/nicely/issues/352)).

## Gotchas

- One conversion of the fixture takes about 1 minute on the macOS test host.
- The dry run scans up to 10 preview frames. Damage that the scan misses shows only in the run.
