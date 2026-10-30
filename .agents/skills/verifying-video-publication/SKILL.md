---
name: verifying-video-publication
description: "Use when changing video-convert staging, publication, overwrite consent, cancellation, cleanup, or the pinned Nicely SDK signal answer."
---

# Verify video publication

The destination must contain a complete worker result, preserve the input, and require `--force` to replace existing output.
A break can destroy previous output or expose partial bytes as finished work.

Copy this checklist and track completion:

- [ ] Confirm the model still describes the changed code
- [ ] Run the proof and its negative controls
- [ ] Interpret every result
- [ ] Commit the current record

1. Compare changed functions with the [coverage record](references/coverage.md) and comments in `verification/publication/Publication.tla`
   Confirm that context sampling, destination checks, and publication remain separate actions
   Update the mapping before interpreting a pass when their order changes
2. Check prerequisites with `go version`, `git --version`, `just --version`, and `mise --version`
   Use Linux for the inotify probe
   If a tool is absent, follow its [installation instructions](references/tools.md)
3. Run `just verify-publication` from the repository root
   The runner installs the pinned managed Java through `mise exec` and verifies the downloaded TLC jar's SHA-256
   It copies the current tree to scratch, runs nine configurations, probes the executable, and tests a real-code mutation
   Completion requires JSON with `"ok":true` and the record path
4. Read every TLC result with the pstack `/statechart` skill
   Use its shortest path for each violation and its state transition chart for a pass
   If unavailable, read the action sequences in the retained logs and report the missing interpretation tool
   The runner prints the logs directory on stderr
   Read the [result guide](references/results.md) to classify each result
5. For an unexpected result, compare its trace with the actual syscall order
   Reproduce the boundary on the executable when the trace and code disagree
   Keep a real product violation as a finding
   Correct a modeling error, then rerun the proof
   Change an expected result only after an executable check demonstrates the intended behavior
6. After intended changes, update the model, configurations, probe, and coverage record together
   Advance the coverage synchronization commit only after checking every mapped action against that commit
   Run `just verify-publication` again, then `just check`
   Commit `verification/publication/record.json` when both pass

The normal gate needs no Java
It rejects absent records and changes to covered source, SDK files, model, configurations, probe, runner, or tool pins
The record hashes the current bytes, including uncommitted changes
Its source commit identifies the reviewed base, while its hashes identify the verified tree

For drift that skipped verification, use `/maintain-formal-verification-skill` to reconcile the model and code
