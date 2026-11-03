# Adoption corrections

Each row records a correction found while adopting the verification loop of Nicely [#427](https://github.com/pascalandy/nicely/issues/427). Owner and level follow D068. Gate times and session IDs belong in the pull request.

| ID | Finding | Real defect | Owner | Level | Correction and proof |
|---|---|---|---|---|---|
| C1 | Dry run omits the unchecked publication operation | Yes | C, extension | 2, public scenario | `publication.txtar` failed on missing `publication.link`; both create and force now report the operation, and denied syscalls fail closed |
| C2 | A converted MP4 retains private author metadata | Yes | C, extension | 2, made-media case | `reads` found Ana Lima in the output; HandBrake now gets `--no-metadata`. Rotated made media passes independent ffprobe references, including normalization and no-normalization paths |

The update's stale proof and unfilled template forms were expected adoption work, not product defects. Moving the TLC evaluator into `sdk/verification` adopts the existing owner A implementation. This build found no new correction owned by A or B.
