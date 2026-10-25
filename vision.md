# video-convert vision

`ncly video convert` converts one local video into one MP4 for humans and agents, while preserving its input and publishing only a complete output.

## Wraps

- HandBrakeCLI, the local executable from [HandBrake](https://handbrake.fr/), tested with HandBrakeCLI 1.11.2
- The built-in `Fast 1080p30` CPU preset, with the MP4 container selected explicitly

## Refuses

- Overwriting an output without explicit `--force`
- Changing the input, including when the output is a link to it
- Publishing a failed, empty, or interrupted conversion
- Other video jobs, records, batches, progress display, or arbitrary HandBrake arguments
