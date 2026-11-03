# Changelog

## v0.2.2

- Adopt Nicely's verification loop with typed intake and review records, declared publication cases and exact-commit signoff
- Report the unchecked publication operation in dry-run pending_checks
- Run the publication model through the SDK, retaining its nine model outcomes and three executable controls
- Verify existing destinations, links, temporary paths, display rotation, audio and H.265 with independent readers

The default conversion keeps its existing metadata behavior. Output can retain private author, device, capture date and GPS metadata. This is a documented privacy limit, not a metadata-removal feature. [Issue #18](https://github.com/pascalandy/ncly-video-convert/issues/18) tracks opt-in removal, including the HandBrake 1.10 capability requirement and capture-date trade-off. Existing HandBrake support remains unchanged.
