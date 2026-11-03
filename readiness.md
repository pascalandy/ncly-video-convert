Commit: 125c101001944a9aa4291d6fa0700cfe2d9c208f
Reviewer: GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a302-b36b-7983-be12-9f3ccc38b45f; independent reviewer who wrote none of the reviewed code
Verdict: Needs work

# Readiness review

`just records` writes this file from `review.toml`. Change that file, then run `just records` again.

## Findings

### R1-F1 Metadata stripping breaks the supported older HandBrake install

- **Severity**: `block`
- **State**: `open`
- **Spec**: `old-handbrake`
- **Family**: Tools installed
- **Evidence**: convert.go:334 unconditionally passes --no-metadata; chooseEncoder at lines 240-260 checks encoders but not that option. The Ubuntu job 126161508871 of run 41706044940 installs HandBrake 1.7.2+ds1-1build2, passes the dry run, then fails acceptance.txtar:9 with RUNTIME and no output. An independent isolated probe at this head reproduces dry-run exit 0 followed by run exit 1 when a shim omits this option from help and returns 0 without output on the option. The same made input converts with the host HandBrake. Raw commands and answers are in the readiness-round1 evidence copy.
- **Command**: ncly video convert, run and dry run
- **Consequence**: A previously supported installed tool cannot convert any video. The successful preview and the free-space hint misdirect the caller. Hosted Ubuntu checks fail.
- **Owner**: Extension author
- **Fix**: In round 2 apply the Head Chef's relayed v0.2.2 policy by removing --no-metadata and preserving existing metadata behavior. Prove conversion and preview on the supported older tool as well as the current tool. Add no minimum version or new CI pin. The policy has not been applied at this reviewed head.

### R1-F2 Three Output acceptance labels lack their case preconditions

- **Severity**: `block`
- **State**: `open`
- **Spec**: `publication-cases`
- **Family**: Complete private staging
- **Evidence**: testdata/script/acceptance.txtar:13-15 repeat the ordinary fresh conversion's ffprobe read for Exists, without --force; Is a link to the input; and Temporary file. No existing destination, output link, or planted temporary-path link precedes these reads. convert.txtar:58-63 and 165-174 exercise existing destinations and output links with stubs, but do not supply the required domain-reader acceptance cases; no scenario constructs the temporary-path case. Pinned generic-cases.md and acceptance-cases.md explicitly require a made domain file and independent check for each applicable Output case. Local just check passes, demonstrating that label coverage alone misses this gap.
- **Command**: ncly video convert, output refusal and staging acceptance
- **Consequence**: The coverage report counts three safety cases as proven without exercising their preconditions. Regressions in those cases can pass these acceptance reads. This finding does not claim that the current publication implementation writes through a link.
- **Owner**: Extension author
- **Fix**: Construct each named case before its public call. Independently read the preserved destination or input against an authored reference and a bad variant that represents that case's regression. Include accepted and refused inputs for the staging family, and retain the destination and input-link families' own meaningful checks.

### R1-F3 No-normalize conversions replace the source capture time

- **Severity**: `warn`
- **State**: `open`
- **Spec**: `determinism`
- **Family**: Codec and encoder
- **Evidence**: convert.go:334 adds --no-metadata. In an independent made-media probe, the source creation_time is 2026-01-01T00:00:00Z, but two x264 --no-normalize conversions at this head have creation_time values 2026-11-03T20:08:42Z and 2026-11-03T20:08:45Z and different SHA-256 values. A direct HandBrake control without this flag preserves the source creation_time. This contradicts qa.md:67's statement that HandBrake copies the source tag. A further public conversion with normalization has no creation_time tag, and the real-fixture default acceptance passes with matching checksums. Therefore a default macOS CPU checksum failure is not established by this review; the observed loss and clock-dependent files are on the no-normalize path.
- **Command**: ncly video convert, repeated CPU conversions
- **Consequence**: With --no-normalize the output loses the source capture timestamp and changes its file checksum between runs in different seconds, contrary to the QA account of timestamp preservation. The normalized default path must be judged separately.
- **Owner**: Extension author
- **Fix**: Apply the approved metadata-preservation policy in round 2 and prove the declared capture-time and determinism behavior on repeated conversions. Reconcile QA and its executable check with the resulting behavior. Nothing outside this frozen head counts as a fix.

### R1-F4 Real-input acceptance omits the private-metadata promise

- **Severity**: `warn`
- **State**: `open`
- **Spec**: `private-metadata`
- **Family**: Display declarations and private metadata
- **Evidence**: spec.md:69 promises that display metadata is applied while private author metadata is removed. acceptance.sh runs decode, orientation, duration, loudness, determinism, HEVC, and still-image checks, but never reads or rejects private metadata. Its made input has only a fixed creation_time, and its real-fixture branch adds no metadata assertion. The separate made-media scenario's Ana Lima check does not check the real fixture. AGENTS.md requires independent acceptance checks in acceptance.sh on the real input.
- **Command**: ncly video convert, real-input acceptance
- **Consequence**: The real-input acceptance gate can pass when its output violates the current metadata promise. The announced metadata-preservation policy has no accepted-limit row at this head.
- **Owner**: Extension author
- **Fix**: In round 2 apply the Head Chef's relayed privacy policy as an explicit accepted limit covering author, capture date, and GPS, and reconcile the real-input acceptance procedure with that declared policy. Keep this finding open until the author records the policy and a fresh reviewer assesses it. Do not claim that this head has already adopted the limit.

### R1-F5 Real-input acceptance omits publication pending checks

- **Severity**: `warn`
- **State**: `open`
- **Spec**: `publication-pending`
- **Family**: Publication capability is pending
- **Evidence**: spec.md:68 requires publication.link without force and publication.rename with force in dry-run pending_checks. acceptance.sh has no dry-run call and never checks pending_checks. publication.txtar proves both answers with a shell stub, and independent made-media probes at this head return both correct operations, but neither supplies the acceptance.sh check on the real input required by AGENTS.md.
- **Command**: ncly video convert --dry-run, with and without --force
- **Consequence**: Real-input acceptance passes without verifying a required preview property. This is a missing gate, not an observed failure of the current dry-run answer.
- **Owner**: Extension author
- **Fix**: On the real fixture run and inspect both dry-run modes before their corresponding writes. Assert the destination, overwrite choice, publication.link or publication.rename, and lack of dry-run effects. Retain a negative control that rejects an omitted or wrong operation.

### R1-F6 The made-media orientation reader accepts a residual rotation

- **Severity**: `warn`
- **State**: `open`
- **Spec**: `orientation`
- **Family**: Display orientation
- **Evidence**: The ffprobe reads at testdata/script/acceptance.txtar:10-15 request codec, coded dimensions, and decoded frame count, but no display side-data rotation and no displayed pixels. An independent probe copied a successful 96x160, 120-frame output and added a residual rotation of -180 degrees. The existing reader prints exactly the same reference for both files, while a displayed-frame comparison gives SSIM 0.120743. The current output is correctly oriented; the demonstrably wrong output passes this orientation reader.
- **Command**: ncly video convert, made-media orientation acceptance
- **Consequence**: The required made-media check can accept a video displayed upside down, despite the orientation promise. The real-input SSIM check does not repair this scenario's incomplete oracle.
- **Owner**: Extension author
- **Fix**: Make the independent reference account for display rotation, using display side data or decoded displayed pixels. Include a bad variant with correctly sized pixels and a residual display matrix, and prove that the reader rejects it while accepting the correct conversion.

### R1-F7 Declared cleanup limits have no typed acceptance record

- **Severity**: `warn`
- **State**: `open`
- **Spec**: `cleanup-limits`
- **Family**: Stop and cleanup
- **Evidence**: The first independent just finish run exits 1 and reports spec.md:57: this limit has no spec ID, no state limit, and no acceptance. The paragraph under ## Limits includes cleanup losses for SIGKILL, power loss, filesystem refusal, and a worker exceeding the SDK cleanup budget, together with other operating constraints. It provides neither the ID/State/Limit/Accepted table required by finish-gate.md nor the user's acceptance for those losses. The exact first output is preserved in finish-first.log; the spec was not modified.
- **Command**: ncly video convert, declared cleanup guarantees and finish gate
- **Consequence**: The readiness gate rejects this head, and no typed record distinguishes supported behavior from a loss that the user has accepted.
- **Owner**: Extension author
- **Fix**: Reconcile the prose Limits paragraph with the pinned contract. Give each actual accepted loss a stable spec ID, state limit, and the real user's acceptance. Put ordinary supported behavior in the applicable spec sections and prove it. Cover the cleanup family with accepted and refused cases. Do not fabricate consent or weaken the guarantees merely to make finish pass.

## Rounds

| Round | Head | Change review | Change commits | Readiness review | Finish |
|---|---|---|---|---|---|
| 1 | 125c101001944a9aa4291d6fa0700cfe2d9c208f | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a302-b36b-7983-be12-9f3ccc38b45f | 708ad0a0c58bfe99f1dd9764cd1bece2f4b6b16a..125c101001944a9aa4291d6fa0700cfe2d9c208f | GPT-6.1-Sol via Codex; CODEX_THREAD_ID=01a1a302-b36b-7983-be12-9f3ccc38b45f | `pending` |

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
