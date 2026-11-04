Commit: 1c8e0e8a6bf9e70f122945a55f569116375b2e0d
Reviewer: GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a758-936e-7a61-9988-736f31049c78; independent post-update change and readiness reviewer who wrote none of the reviewed implementation
Verdict: Ready with stated limits

# Readiness review

`just records` writes this file from `review.toml`. Change that file, then run `just records` again.

## Findings

### R1-F1 Metadata stripping breaks the supported older HandBrake install

- **Severity**: `block`
- **State**: `fixed`
- **Spec**: `old-handbrake`
- **Family**: Tools installed
- **Commit**: 7c17b027e8287503368c49f5c8e2730941095ede
- **Evidence**: Round 1's older-tool failure remains valid historical evidence. 129174b removes --no-metadata; 7c17b02 adds legacy-handbrake.txtar, whose old parser exits zero without output if that flag returns. The current Tools installed family pairs successful conversion with missing-tool refusals. Independent just check at the reviewed head passes these cases and the real made-media acceptance. Retained ubuntu-v0261-verify logs at 0777195 show full check, generated acceptance and full-fixture acceptance passing with HandBrake 1.7.2+ds1-1build2. All eight source bindings in v0261-tested-source.json match this reviewed tree.
- **Command**: ncly video convert, run and dry run
- **Consequence**: At round 1, a supported older tool could not convert video despite a successful preview, and the hosted Ubuntu acceptance failed. Final-candidate hosted checks remain pending.
- **Owner**: Extension author
- **Fix**: Applied without a new HandBrake minimum or stripping flag. Metadata remains subject to the accepted private-metadata limit.

### R1-F2 Three Output acceptance labels lack their case preconditions

- **Severity**: `block`
- **State**: `fixed`
- **Spec**: `publication-cases`
- **Family**: Complete private staging
- **Commit**: 792fe980bce9548d2ec5c617a8905af5b0c8202e
- **Evidence**: The current acceptance.txtar constructs a distinct existing MP4 before refusal, an output symlink to its input before refusal, and a predictable temporary-path symlink before conversion. Each case protects its files and independently reads preserved content against authored references and meaningful bad variants. c63cd5f also reads the separately published temporary-case output. Publication scenarios pair success with failed encode, refused link or rename, and concurrent destination creation. 792fe98 supplies the corresponding real-input preconditions in acceptance.sh. Independent just check at the reviewed head and the retained full-fixture logs pass.
- **Command**: ncly video convert, output refusal and staging acceptance
- **Consequence**: At round 1, three safety cases counted as proven without their preconditions, allowing regressions to pass those readers. No implementation write through a link was established.
- **Owner**: Extension author
- **Fix**: Applied across the existing-destination, input-alias and private-staging families.

### R1-F3 No-normalize conversions replace the source capture time

- **Severity**: `warn`
- **State**: `limit`
- **Spec**: `private-metadata`
- **Family**: Descriptive metadata can remain
- **Evidence**: The observed round-1 no-normalize timestamp loss is not relabeled a passing preservation test. The Head Chef's accepted private-metadata limit promises neither retention nor removal of capture dates. spec.md and qa.md now distinguish container timestamps from media determinism and document the older HandBrake conversion-time behavior. acceptance.txtar compares repeated no-normalize frame counts, duration, audio packet hashes and video SSIM. acceptance.sh checks repeats in the default normalization mode. Both have independent readers; the made-media repeat has a shortened bad variant. Current check and retained om1 and Ubuntu real-fixture runs pass those media criteria. No default macOS CPU checksum failure is inferred.
- **Command**: ncly video convert, repeated CPU conversions
- **Consequence**: Capture dates and container checksums can vary by input, tool build and normalization. The accepted policy permits that variation while media determinism remains tested separately.
- **Owner**: Extension author
- **Fix**: Closed by the explicitly accepted private-metadata limit, not by a universal timestamp-preservation claim.

### R1-F4 Real-input acceptance omits the private-metadata promise

- **Severity**: `warn`
- **State**: `limit`
- **Spec**: `private-metadata`
- **Family**: Display declarations
- **Evidence**: The accepted private-metadata row describes author, capture date, device and location retention, with both exact privacy Output bindings. Both applicable matrix rows name its ID. The two reads -case ... -limit private-metadata calls independently check upright displayed content and the complete retained input-artist reference. Their raw rotated-input and missing-artist bad variants fail. No text is promised absent by these cases, so no -private exemption is invented. Real acceptance reads actual source and target tags for each codec and permits zero matches; a normalized artist control proves possible retention when needed. Retained v0.26.1 om1 logs report 1/2 known values per codec; Ubuntu reports 0/2 and passes the control. C20's older universal-retention assertion remains failed. The same committed ffprobe error-73 control was swallowed before 0ce0867 and is rejected afterward, including the v0.26.1 replay.
- **Command**: ncly video convert, real-input acceptance
- **Consequence**: Private metadata can remain in shared outputs. The accepted limit discloses this loss of privacy; real-input acceptance no longer promises universal stripping or retention.
- **Owner**: Extension author
- **Fix**: Closed under the Head Chef's accepted limit and the v0.26.1 accepted-limit protocol. Opt-in stripping remains issue #18.

### R1-F5 Real-input acceptance omits publication pending checks

- **Severity**: `warn`
- **State**: `fixed`
- **Spec**: `publication-pending`
- **Family**: Publication capability is pending
- **Commit**: 792fe980bce9548d2ec5c617a8905af5b0c8202e
- **Evidence**: acceptance.sh now runs create and forced-replace previews before their writes. It checks exact video_decode plus publication.link or publication.rename, overwrite consent, absent create output, preserved replacement output and no stage. The planned path supplies the planted temporary-link case, and the published answer is independently checked. publication.txtar retains predicts for both modes and paired without-link and without-rename refusals. Independent just check at the reviewed head and the retained v0.26.1 real-fixture acceptance logs pass.
- **Command**: ncly video convert --dry-run, with and without --force
- **Consequence**: At round 1, real-input acceptance omitted a required preview property. The observed dry-run answers themselves were correct.
- **Owner**: Extension author
- **Fix**: Applied to real-input acceptance and retained public-boundary publication tests.

### R1-F6 The made-media orientation reader accepts a residual rotation

- **Severity**: `warn`
- **State**: `fixed`
- **Spec**: `orientation`
- **Family**: Display orientation
- **Commit**: 9a9b27d3063f1d7383a3c0e4832154e7261b12b0
- **Evidence**: upright.sh refuses residual display rotation and compares decoded frames through ffprobe's independent lavfi reader against the authored cclock transpose of flat.mp4. acceptance.txtar supplies both a residual 180-degree matrix on correctly sized pixels and pixels flipped on both axes as separate bad variants. The correct rotated conversion passes and both bad variants are rejected by independent just check. The Display orientation family also retains the still-image refusal pair. Real-input acceptance compares displayed source and output pixels; its retained om1 and Ubuntu runs pass.
- **Command**: ncly video convert, made-media orientation acceptance
- **Consequence**: At round 1, an upside-down output could pass the made-media reader despite the orientation promise. A separate real-input check did not close that scenario gap.
- **Owner**: Extension author
- **Fix**: Applied with independent display-pixel and residual-matrix checks.

### R1-F7 Declared cleanup limits have no typed acceptance record

- **Severity**: `warn`
- **State**: `fixed`
- **Spec**: `cleanup-limits`
- **Family**: Stop and cleanup
- **Commit**: 07771955fa419697498144867aa0d38ba6419d0c
- **Evidence**: The original finish-first.log remains a failed round-1 result. The current spec separates Runtime constraints from typed accepted limits and proves cleanup-limits in Acceptance for supported hosts. The Stop and cleanup family pairs ordinary success with INT and TERM during preparation and encoding, including the unchanged 20,000-file resistant worker. Byte-identical f0dc93a/v0.26.0 and bfe783d/v0.26.1 cases respectively fail stage assertion 91 and pass. Current independent just check passes. All 45 publication input hashes match, including v0.26.1 core dispatch, non-test runner and process-tree sources; independent executable publication probes also pass. Source-candidate 7aae38d has ten H and ten installed P readings under the approved external driver, all clean. The older 4b2de72 baseline used H under the Go harness driver; these drivers are not identical. The same new extension still leaks on source-built old cores, expressly covered by the accepted core-minimum limit and nicely#484. Normal check accepts that compatibility limit without an Output binding. Final frozen-head om1 and Mini sets, hosted checks and exact-SHA signoff remain pending manual gates, not evidence claimed by this review.
- **Command**: ncly video convert, declared cleanup guarantees and finish gate
- **Consequence**: Round 1 failed readiness because cleanup bounds lacked typed disposition. Supported-host cleanup is now proven; older-core enforcement remains an accepted compatibility limit and final-head release gates remain pending.
- **Owner**: Extension author
- **Fix**: Applied through the v0.26.1 update, supported-host cleanup tests and explicit accepted core-minimum limit. D068 step 3 for minimum-version enforcement waits for a future Nicely release. Final host, hosted and signoff gates are not waived.

## Rounds

| Round | Head | Change review | Change commits | Readiness review | Finish |
|---|---|---|---|---|---|
| 1 | 125c101001944a9aa4291d6fa0700cfe2d9c208f | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a302-b36b-7983-be12-9f3ccc38b45f | 708ad0a0c58bfe99f1dd9764cd1bece2f4b6b16a..125c101001944a9aa4291d6fa0700cfe2d9c208f | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a302-b36b-7983-be12-9f3ccc38b45f | `fail` |
| 2 | 1c8e0e8a6bf9e70f122945a55f569116375b2e0d | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a758-936e-7a61-9988-736f31049c78 | 125c101001944a9aa4291d6fa0700cfe2d9c208f..1c8e0e8a6bf9e70f122945a55f569116375b2e0d | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a758-936e-7a61-9988-736f31049c78 | `pending` |

## Order of round 1

| Step | Commit |
|---|---|
| Intake | 9babff870139812e1b96da0ad3a30d31e0f4110c |
| Spec | 3ab2b7ce41d9ae090c68de65f4acd7d03c66a285 |
| Scenario | 62161e4793734690c7c2d785f46297ba05f827b2 |
| Handler | db5f124e8033dfc2d3af2c77c592c36431817188 |

## First failing run

- **Commit**: 62161e4793734690c7c2d785f46297ba05f827b2
- **Command**: `just scenario publication`
- **Exit**: 1
- **Assertion**: stdout '"pending_checks":\["video_decode","publication.link"\]'
