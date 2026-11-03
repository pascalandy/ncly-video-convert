# Adoption corrections

Each row records a correction found while adopting the verification loop of Nicely [#427](https://github.com/pascalandy/nicely/issues/427). Owner and level follow D068. Gate times and session IDs belong in the pull request.

| ID | Finding | Real defect | Owner | Level | Correction and proof |
|---|---|---|---|---|---|
| C1 | Dry run omits the unchecked publication operation | Yes | C, extension | 2, public scenario | `publication.txtar` failed on missing `publication.link`; both create and force now report the operation, and denied syscalls fail closed |
| C2 | Private metadata retention was undocumented | Yes | C, extension | 4, policy and consent | Scenario found Ana Lima in the output. Head Chef retained the existing metadata behavior for v0.2.2, accepted the `private-metadata` limit, and requested opt-in follow-up [#18](https://github.com/pascalandy/ncly-video-convert/issues/18). Independent cases check the declared retention and display behavior |
| C3 | A draft strip change broke HandBrake before 1.10 while dry run passed | Yes | C, extension | 2, hosted media case | Ubuntu CI exposed the unsupported flag. The policy correction removes it, retaining the old help/build compatibility; the opt-in follow-up must fail closed when stripping is requested on unsupported builds |
| C4 | Output-case labels had no corresponding setup | Yes | C, extension | 2, made-media cases | The accepted and refused calls now create an existing output, an output symlink, and a predictable temporary symlink before reading the preserved or published media |
| C5 | Real acceptance omitted pending-operation and metadata checks | Yes | C, extension | 2, real-input acceptance | Independent ffprobe checks retained metadata, and both dry-run modes report their pending publication operation |
| C6 | Orientation reader omitted the residual display matrix | Yes | C, extension | 2, independent reader | The reader now includes side-data rotation as well as decoded frame count and dimensions |
| A1 | Accepted privacy limits cannot satisfy output-case coverage | Yes | A, kit | 2, evaluator | [nicely#462](https://github.com/pascalandy/nicely/issues/462) records the input, host, failure and required regression. The SDK limit is explicit in spec.md. The extension waits for the Nicely release and will update and rerun its gates |
| C7 | Prose cleanup bounds did not fit the typed limit form | Yes | C, extension | 2, spec and scenarios | The main cleanup promise now states its existing filesystem and ordinary-return conditions; runtime constraints remain explicit, and cleanup has an acceptance line with success and refusal coverage. No new cleanup exception or consent is invented |
| C8 | Timestamp assertion silently required HandBrake 1.10 | Yes | C, extension | 2, distro scenario | Actual Ubuntu 1.7.2 fails the assertion while the newer host passes. The criterion now compares decoded media and audio packet hashes; sourced runtime documentation states the version-dependent clock behavior |

The update's stale proof and unfilled template forms were expected adoption work, not product defects. Moving the TLC evaluator into `sdk/verification` adopts the existing owner A implementation. The accepted-limit evaluator defect is owned by A and follows D068 through nicely#462.
