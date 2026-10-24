# Nicely video convert

Convert one local video with HandBrakeCLI:

```sh
ncly video convert clip.mov --dry-run
ncly video convert clip.mov
```

The result is `clip.converted.mp4` beside the input, using HandBrake's built-in `Fast 1080p30` preset. The command preserves the original and publishes the output only after HandBrake succeeds and produces a nonempty regular file.

If the output already exists, review it before replacing it:

```sh
ncly video convert clip.mov --force --dry-run
ncly video convert clip.mov --force
```

A failed conversion keeps the previous output. Errors, timeouts, Ctrl-C, and SIGTERM clean staging data. A dry run checks paths and HandBrake availability without encoding. It cannot establish that the video will decode.

## Install

Install Go, just, and HandBrakeCLI, then run from this repository:

```sh
go mod tidy
just check
just install
ncly describe video convert --json
```

Use `ncly video convert --help` for options. Machine callers use `--json --no-input`; successful JSON includes the verified absolute `output` path and `size_bytes`. The default timeout is two hours.

The command has no batches, records, or progress display. See [spec.md](spec.md) for failures and filesystem limits, and [AGENTS.md](AGENTS.md) to change the extension.
