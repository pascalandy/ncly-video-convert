---
name: video-convert
description: Use when converting one local video to MP4 with HandBrakeCLI while preserving the original, in H.264 or H.265, with the audio at -16 LUFS.
---

# Convert one video

Use `ncly video convert` for one video and one MP4 output. It encodes on the GPU when the machine has one, and brings the audio to -16 LUFS. For batches, other video jobs, or custom encoding settings, choose another tool.

Keep the default H.264 unless the user asks for H.265, or for HDR to stay HDR: then pass `--codec h265`. Pass `--no-normalize` only when the user wants the loudness of the input.

1. Read `ncly describe video convert --json`, then dry-run the intended call
2. Resolve any prerequisite, input, or destination error before execution. A dry run checks availability, paths, and that HandBrake finds a video in the input. Only the conversion decodes the whole video
3. Run the conversion. Success returns a verified `output` and `size_bytes`, with the `codec`, the `encoder`, and whether the audio was `normalized`. The input stays unchanged
4. A warning keeps the output. On `VIDEO_CONVERT_GPU_RUNTIME_MISSING`, relay its hint to the user, who can install Intel's media runtime. On `VIDEO_CONVERT_GPU_FAILED`, the GPU encoder failed and the command ran the CPU encoder instead
5. On failure, stdout is empty and the error object ends stderr. Read the code there
6. On `VIDEO_CONVERT_UNREADABLE`, the input is not a readable video, such as a text, audio-only, or truncated file, or a still image. Report the file named in the hint instead of repeating the call
7. On `CONFIRMATION_REQUIRED`, review the existing output before explicitly consenting to its replacement. On failure or interruption, inspect the error before repeating the call. A failed conversion preserves an earlier output
