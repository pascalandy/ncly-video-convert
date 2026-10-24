# video-convert spec

`manifest.toml` holds the command's flags, effects, requirements, and result schemas.

## ncly video convert

Success means HandBrakeCLI exited zero and the published output exists as a nonempty regular file. The answer names its absolute path and size. `clip.mov` produces `clip.converted.mp4` in the input folder. The command uses `Fast 1080p30` and the MP4 container.

The command validates a readable regular input, the destination, and HandBrakeCLI availability in both execution and dry run. A dry run starts only HandBrake's harmless version command. It creates no staging folder or output and reports the planned paths and preset. Video decoding remains a pending check until execution.

Execution writes in a private staging folder beside the destination. Only a successful, nonempty regular output can be published. Without overwrite consent, an atomic hard link refuses a destination created during conversion. With consent, an atomic rename replaces the previous output after success. A final stat proves the published file exists. All return paths remove staging, after the SDK stops HandBrake on a timeout or signal. A failed forced conversion preserves the previous output.

The input is opened only for reading. An output that is a symlink, directory, special file, or the same file as the input is refused even with overwrite consent. The command never edits the existing output in place.

## Effects

The sole user write creates or replaces the converted MP4. No network, paid service, key, reusable partial work, or run record is needed.

## Failures

| Condition | Code |
|---|---|
| Input missing, unreadable, or not regular; timeout invalid; destination unsafe | `USAGE_INVALID` |
| Output exists without overwrite consent, including an arrival during conversion | `CONFIRMATION_REQUIRED` |
| HandBrakeCLI missing | `PREREQ_MISSING` |
| HandBrake exits nonzero, times out, produces no nonempty regular output, or publication or cleanup fails | `RUNTIME` |
| Ctrl-C or SIGTERM | `INTERRUPTED` or `TERMINATED` |

## Limits

HandBrake must support the built-in preset. Availability is checked without parsing its version because distribution builds can print a source revision instead of a release number. The destination filesystem must support hard links for publication without overwrite consent. Dry run does not decode the video or reserve a destination. Cleanup handles ordinary errors, timeouts, Ctrl-C, and SIGTERM; SIGKILL, power loss, and filesystem refusal can prevent any process from cleaning temporary files. Such temporary data is never published as the output. HandBrake inherits ordinary environment variables under the SDK's environment rules.
