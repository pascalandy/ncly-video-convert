# Nicely video convert

Convert one local video with HandBrakeCLI:

```sh
ncly video convert clip.mov --dry-run
ncly video convert clip.mov
```

The result is `clip.converted.mp4` beside the input, using HandBrake's built-in `Fast 1080p30` preset. The command encodes on the GPU when the machine has one that HandBrake can use: Apple VideoToolbox on a Mac, or Intel Quick Sync on Linux. Otherwise it encodes on the CPU. It brings the audio to -16 LUFS with ffmpeg. The command preserves the original and publishes the output only after HandBrake succeeds and produces a nonempty regular file.

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

A failed conversion keeps the previous output. Errors, timeouts, Ctrl-C, and SIGTERM clean staging data. A dry run checks the paths, the output folder's access, HandBrake availability, and that HandBrake finds a video in the input, without encoding. It cannot establish that the whole video will decode.

## Install

Install [Nicely](https://github.com/pascalandy/nicely#install) v0.23.0 or later, so that `ncly` is on your PATH. An older `ncly` skips this named extension.

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

The command has no batches, records, or progress display. See [spec.md](spec.md) for failures and filesystem limits, and [AGENTS.md](AGENTS.md) to change or release the extension.

## Let your agent find it

Add one line to your agent instructions, such as `AGENTS.md` or `CLAUDE.md`, to teach an agent to use Nicely:

```text
When asked to use nicely, run ncly skill view nicely
```
