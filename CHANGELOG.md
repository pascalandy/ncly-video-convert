# Changelog

## v0.2.2

- Adopt Nicely's verification loop with typed intake and review records, declared publication cases and exact-commit signoff
- Report unchecked `publication.link` or `publication.rename` in dry-run `pending_checks`
- Run the publication model through the SDK, retaining its nine model outcomes and three executable controls
- Verify existing destinations, links, temporary paths, display rotation, audio and H.265 with independent readers

The command does not request descriptive metadata removal. Author, device, capture date or location data can remain, depending on the input, tool versions and normalization. Neither retention nor removal is guaranteed. [Issue #18](https://github.com/pascalandy/ncly-video-convert/issues/18) tracks opt-in removal, including the HandBrake 1.10 capability requirement and capture-date trade-off. Existing HandBrake support remains unchanged.
