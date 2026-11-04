# Nicely video convert

Convert one local video with HandBrakeCLI:

```sh
ncly video convert clip.mov --dry-run
ncly video convert clip.mov
```

The result is `clip.converted.mp4` beside the input, using HandBrake's built-in `Fast 1080p30` preset. The command encodes on the GPU when the machine has one that HandBrake can use: Apple VideoToolbox on a Mac, or Intel Quick Sync on Linux. Otherwise it encodes on the CPU. It brings the audio to -16 LUFS with ffmpeg. The command preserves the original and publishes the output only after HandBrake succeeds and produces a nonempty regular file.

The command does not request descriptive metadata removal. The output can keep the input's author, capture date, device metadata or location, depending on the input, tool versions and normalization. Do not rely on either retention or removal. Inspect metadata before sharing a converted video:

```sh
ffprobe -v error -show_entries format_tags -of default=noprint_wrappers=1 clip.converted.mp4
```

An opt-in stripping flag is tracked in [#18](https://github.com/pascalandy/ncly-video-convert/issues/18).

The output is H.264, which plays everywhere. For H.265 in 10 bits, which keeps an HDR video in HDR, or to keep the loudness of the input:

```sh
ncly video convert clip.mov --codec h265
ncly video convert clip.mov --no-normalize
```

If the output already exists, review it before replacing it:

```sh
ncly video convert clip.mov --force --dry-run
ncly video convert clip.mov --force
```

A failed conversion keeps the previous output. With Nicely v0.26.1 or later, the command removes staging after errors, timeouts, SIGINT and SIGTERM, including a HandBrake worker that ignores the stop signal. This fixes [nicely#468](https://github.com/pascalandy/nicely/issues/468).

A dry run checks paths, folder access, HandBrake availability, and the input's video title without encoding. It cannot prove that the whole video decodes or that the folder supports publication. Without `--force`, a new output needs a hard link. With `--force`, publication uses a rename whether or not the output already exists. The JSON answer lists these pending operations as `video_decode` and either `publication.link` or `publication.rename` in `pending_checks`.

## Install

Install [Nicely](https://github.com/pascalandy/nicely#install) v0.26.1 or later, so that `ncly` is on your PATH. Upgrade the host with the extension: the host supplies the cleanup fix for a worker that ignores a stop signal. The manifest cannot enforce this minimum yet ([nicely#484](https://github.com/pascalandy/nicely/issues/484)). Older cores can leave a staging folder after SIGTERM; the input and previous output stay safe.

Also install Go, just, HandBrakeCLI, and ffmpeg, such as `brew install handbrake ffmpeg` on macOS or `sudo pacman -S handbrake-cli ffmpeg` on Arch Linux. On Linux with an Intel GPU, Quick Sync also needs Intel's media runtime, such as `sudo pacman -S vpl-gpu-rt` on Arch Linux. Without it, the command encodes on the CPU and warns. Replace TAG with the highest release tag, then build the extension:

```sh
git clone --branch TAG https://github.com/pascalandy/ncly-video-convert
cd ncly-video-convert
just install
ncly describe video convert --json
ncly skill view video-convert
```

`git ls-remote --tags --refs --sort=-v:refname https://github.com/pascalandy/ncly-video-convert` lists the tags, highest first. For an annotated tag, Git can warn that the tag `is not a commit!`, then note a detached HEAD. The clone holds the tagged commit.

`ncly describe` confirms the install with `video convert` and the source `extension`. `just install` puts the extension into Nicely's extensions folder, in place of an earlier install, and grants no key. `just uninstall` in the clone removes it. An agent reads `ncly skill view video-convert` to learn when to use the extension.

Use `ncly video convert --help` for options. Machine callers use `--json --no-input`; successful JSON includes the verified absolute `output` path, `size_bytes`, `codec`, `encoder`, and `normalized`. The default timeout is two hours.

The extension speaks English only. Under another language, such as `--lang fr-CA`, its help and errors fall back to English without a warning.

The command has no batches, records, or progress display. See [spec.md](spec.md) for failures, accepted limits and runtime constraints, and [AGENTS.md](AGENTS.md) to change or release the extension.

## Let your agent find it

Add one line to your agent instructions, such as `AGENTS.md` or `CLAUDE.md`, to teach an agent to use Nicely:

```text
When asked to use nicely, run ncly skill view nicely
```
