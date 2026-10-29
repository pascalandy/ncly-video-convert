---
name: video-convert
description: Use when converting one local video to MP4 with HandBrakeCLI while preserving the original.
---

# Convert one video

Use `ncly video convert` for one video and one MP4 output. For batches, other video jobs, or custom encoding settings, choose another tool.

1. Read `ncly describe video convert --json`, then dry-run the intended call
2. Resolve any prerequisite, input, or destination error before execution. A dry run checks availability, paths, and that HandBrake finds a video in the input. Only the conversion decodes the whole video
3. Run the conversion. Success returns a verified `output` and `size_bytes`. The input stays unchanged
4. On failure, stdout is empty and the error object ends stderr. Read the code there
5. On `VIDEO_CONVERT_UNREADABLE`, the input is not a readable video, such as a text, audio-only, or truncated file. Report the file named in the hint instead of repeating the call
6. On `CONFIRMATION_REQUIRED`, review the existing output before explicitly consenting to its replacement. On failure or interruption, inspect the error before repeating the call. A failed conversion preserves an earlier output
