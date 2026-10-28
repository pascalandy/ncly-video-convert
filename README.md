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

A failed conversion keeps the previous output. Errors, timeouts, Ctrl-C, and SIGTERM clean staging data. A dry run checks the paths, the output folder's access, HandBrake availability, and that HandBrake finds a video in the input, without encoding. It cannot establish that the whole video will decode.

## Install

Install [Nicely](https://github.com/pascalandy/nicely#install) v0.23.0 or later, so that `ncly` is on your PATH. An older `ncly` skips this named extension.

Also install Go, just, and HandBrakeCLI, such as `brew install handbrake` on macOS or `sudo pacman -S handbrake-cli` on Arch Linux. Replace TAG with a release tag, such as `v0.1.1`, then build the extension:

```sh
git clone --branch TAG https://github.com/pascalandy/ncly-video-convert
cd ncly-video-convert
just install
ncly describe video convert --json
ncly skill view video-convert
```

`git ls-remote --tags https://github.com/pascalandy/ncly-video-convert` lists the available tags. For an annotated tag, Git can warn that the tag `is not a commit!`, then note a detached HEAD. The clone holds the tagged commit.

`ncly describe` confirms the install with `video convert` and the source `extension`. `just install` puts the extension into Nicely's extensions folder, in place of an earlier install, and grants no key. An agent reads `ncly skill view video-convert` to learn when to use the extension.

Use `ncly video convert --help` for options. Machine callers use `--json --no-input`; successful JSON includes the verified absolute `output` path and `size_bytes`. The default timeout is two hours.

The command has no batches, records, or progress display. See [spec.md](spec.md) for failures and filesystem limits, and [AGENTS.md](AGENTS.md) to change or release the extension.
