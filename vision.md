# video-convert vision

`ncly video convert` converts one local video into one MP4 for humans and agents, while preserving its input and publishing only a complete output.

## Wraps

- HandBrakeCLI, the local executable from [HandBrake](https://handbrake.fr/), tested with HandBrakeCLI 1.11.2
- The built-in `Fast 1080p30` preset, with the MP4 container selected explicitly, and the GPU encoder of the machine when it has one: Apple VideoToolbox or Intel Quick Sync
- The two codecs that play most widely: H.264, the default, and H.265 in 10 bits
- ffmpeg's `loudnorm` filter, which brings the audio to -16 LUFS, as the earlier video tools of the author did

## Intake

`just records` writes this part from `intake.toml`. Change that file, then run `just records` again.

- The command converts one readable regular local file to a nonempty MP4 beside it, using Fast 1080p30; H.264 is the default and H.265 uses Main 10. Source: spec.md, at 708ad0a. Read: ncly video convert, success criteria and encoder selection
- The input is read only. The sole user write is the converted MP4. No network service, key, paid request, reusable run record, or interactive question is needed. Source: spec.md and convert.go, at 708ad0a. Read: Effects; convert, conversionJob.encode, transcode, normalizeAudio, and publish
- Preparation resolves paths, checks access and destination safety, scans one title, selects a host encoder, and probes ffmpeg when audio normalization is needed. A dry run performs that preparation without staging or publishing. Source: convert.go, at 708ad0a. Read: convert, conversionJob.prepare, scanVideo, checkDestination, and canonicalInput
- HandBrakeCLI writes its JSON title set and encode progress to stdout; scan-only mode prints the title set and exits, and an absent title sets HB_ERROR_WRONG_INPUT. Source: https://github.com/HandBrake/HandBrake/blob/1.11.2/test/test.c, at 1.11.2. Read: main, PrintTitleSetInfo, show_progress_json, and HandleEvents HB_STATE_SCANDONE
- HandBrake CLI options depend on host hardware. The adapter reads --help for VideoToolbox and Quick Sync availability and falls back once to the codec's CPU encoder after a GPU encode failure. Source: https://handbrake.fr/docs/en/latest/cli/command-line-reference.html and convert.go, at documentation read 2026-11-03; adapter 708ad0a. Read: Command line reference introduction; chooseEncoder, encoderFor, and transcode
- loudnorm accepts measured loudness, range, threshold and true peak for its linear second pass, and emits JSON statistics; dynamic processing can require 192 kHz, so the adapter explicitly writes 48 kHz AAC. Source: https://github.com/FFmpeg/FFmpeg/blob/n9.0.2/libavfilter/af_loudnorm.c and convert.go, at FFmpeg n9.0.2; adapter 708ad0a. Read: loudnorm_options, config_input, uninit; measureAudio and normalizeAudio
- The two-hour default timeout covers scan, encode and normalization together. The SDK stops children on timeout, SIGINT and SIGTERM. Staging is removed on ordinary returns; SIGKILL, power loss and filesystem refusal can prevent cleanup. Source: spec.md and convert.go, at 708ad0a. Read: execution paragraph and Limits; convert, conversionJob.encode, and its deferred cleanup
- Preparation rejects invalid paths, codecs and deadlines; missing tools return PREREQ_MISSING. HandBrake scan exit 2 and a still-image title return VIDEO_CONVERT_UNREADABLE; other worker failures are RUNTIME. Repeating after an output exists requires --force. Source: spec.md and convert.go, at 708ad0a. Read: Failures; convert, scanVideo, transcode, and checkDestination
- A private staging directory is beside the destination. A complete staged file is published with a hard link in create mode and an atomic replacing rename in force mode; neither operation writes through an output link. Source: convert.go, at 708ad0a. Read: conversionJob.encode, publish, and checkDestination
- Orientation, dimensions, frame rate, sample aspect ratio, colour and HDR declarations affect display. The preset bounds dimensions and frame rate; real acceptance checks orientation and duration, and H.265 keeps ten-bit HDR. Source: spec.md, qa.md, and acceptance.sh, at 708ad0a. Read: encoder paragraph; Result, Fidelity and Known limits; probe, ssim, decoded, heard and hevc-output checks
- Does the dry-run answer disclose that hard-link or replacing-rename capability is still unchecked until publication?. Unknown: see the line `publication-pending` of `spec.md`

**Experiments.**

- `HandBrakeCLI --version; ffmpeg -version`: om1 has HandBrake 20260607133641-9eb6c9368-makepkg and FFmpeg n9.0.2. HandBrake's distribution revision differs from the reference 1.11.2; acceptance must check the actual host output.
- `gh release view -R pascalandy/nicely --json tagName; ncly --version`: Latest published Nicely was v0.26.0 before just update; the verified linux_amd64 release archive installed ncly v0.26.0 on om1.

## Refuses

- Overwriting an output without explicit `--force`
- Changing the input, including when the output is a link to it
- Publishing a failed, empty, or interrupted conversion
- Converting a still image, such as a PNG or a JPEG, which HandBrake reads as a title of one frame
- Other video jobs, records, batches, progress display, or arbitrary HandBrake arguments
