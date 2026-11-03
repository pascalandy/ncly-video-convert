# video-convert vision

`ncly video convert` converts one local video into one MP4 for humans and agents, while preserving its input and publishing only a complete output.

## Wraps

- HandBrakeCLI, the local executable from [HandBrake](https://handbrake.fr/), tested with HandBrakeCLI 1.11.2
- The built-in `Fast 1080p30` preset, with the MP4 container selected explicitly, and the GPU encoder of the machine when it has one: Apple VideoToolbox or Intel Quick Sync
- The two codecs that play most widely: H.264, the default, and H.265 in 10 bits
- ffmpeg's `loudnorm` filter, which brings the audio to -16 LUFS, as the earlier video tools of the author did

## Intake

`just records` writes this part from `intake.toml`. Change that file, then run `just records` again.

- Effects: `worker/inspect.py` reads the file once, in chunks of 64 KiB, and writes nothing. Source: `worker/inspect.py` of the template, at the Nicely version that `go.mod` pins. Read: the function `main`
- Exit codes: 0 with one JSON object on stdout, and 2 for a wrong argument count. Source: `worker/inspect.py` of the template, at the Nicely version that `go.mod` pins. Read: `main` and the last two lines of the file
- Resource use: memory stays near one chunk, whatever the size of the file. Source: `worker/inspect.py` of the template, at the Nicely version that `go.mod` pins. Read: the loop of `main`
- Known failures: Python raises an error and exits 1 on a file that it cannot open. Source: `worker/inspect.py` of the template, at the Nicely version that `go.mod` pins. Read: the call to `open` in `main`, which nothing catches

**Experiments.**

- `python3 worker/inspect.py notes.md`: on a file of three lines, the last without a newline, it printed `"lines": 3` and exited 0

## Refuses

- Overwriting an output without explicit `--force`
- Changing the input, including when the output is a link to it
- Publishing a failed, empty, or interrupted conversion
- Converting a still image, such as a PNG or a JPEG, which HandBrake reads as a title of one frame
- Other video jobs, records, batches, progress display, or arbitrary HandBrake arguments
