# video-convert spec

`manifest.toml` holds the command's flags, effects, requirements, and result schemas.

## ncly video convert

Success means HandBrakeCLI exited zero and the published output exists as a nonempty regular file. The answer names its absolute path and size. `clip.mov` produces `clip.converted.mp4` in the input folder. The command uses `Fast 1080p30` and the MP4 container.

In both execution and dry run, the command validates a readable regular input, the destination and its folder's read, write and search access, HandBrakeCLI availability, and a video title in the input. One scan of the input, `HandBrakeCLI --input <file> --scan`, finds that title. It reads one title and a few preview frames, encodes nothing, and writes no file. A dry run starts only this scan. It creates no staging folder or output and reports the planned paths and preset. Decoding of the whole video remains a pending check until execution.

Execution writes in a private staging folder beside the destination. Only a successful, nonempty regular output can be published. Without overwrite consent, an atomic hard link refuses a destination created during conversion. With consent, an atomic rename replaces the previous output after success. A final stat proves the published file exists. All return paths remove staging, after the SDK stops HandBrake on a timeout or signal. A failed forced conversion preserves the previous output.

The input is opened only for reading. Its folder is resolved through the file system, so `..` after a symlink names the file that the kernel opens, and the output goes beside that file. An output that is a symlink, directory, special file, or the same file as the input is refused even with overwrite consent. The command never edits the existing output in place.

## Effects

The sole user write creates or replaces the converted MP4. No network, paid service, key, reusable partial work, or run record is needed.

## Failures

| Condition | Code |
|---|---|
| Input missing, unreadable, or not regular; timeout invalid; destination unsafe | `USAGE_INVALID` |
| The scan exits 2 because HandBrake finds no video title, such as in a text, audio-only, empty, or truncated file. The message and hint name the file | `VIDEO_CONVERT_UNREADABLE`, exit 2 |
| Output exists without overwrite consent, including an arrival during conversion | `CONFIRMATION_REQUIRED` |
| HandBrakeCLI missing | `PREREQ_MISSING` |
| The output folder cannot be read, written or searched; the scan exits with another nonzero code or times out; the encode exits nonzero, times out, or produces no nonempty regular output; publication or cleanup fails | `RUNTIME` |
| Ctrl-C or SIGTERM | `INTERRUPTED` or `TERMINATED` |

## Limits

HandBrake must support the built-in preset. Availability is checked without parsing its version because distribution builds can print a source revision instead of a release number. The destination filesystem must support hard links for publication without overwrite consent. Dry run decodes only what the scan reads, and reserves no destination. The scan passes a still image, since HandBrake reads it as a one-frame title. HandBrake also exits 2 for an unknown preset, so the scan carries no preset, and an encode that exits 2 stays `RUNTIME`. Cleanup handles ordinary errors, timeouts, Ctrl-C, and SIGTERM; SIGKILL, power loss, and filesystem refusal can prevent any process from cleaning temporary files. So can a HandBrake that ignores a stop signal for the SDK's whole cleanup time, 15 seconds, since ncly then stops the extension too. Such temporary data is never published as the output. HandBrake inherits ordinary environment variables under the SDK's environment rules. The command discards HandBrake's stdout, where HandBrake prints encode progress, so the SDK's capture limit does not apply to it.
