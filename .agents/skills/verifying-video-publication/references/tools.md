# Tools

Use Linux for the executable probe
The normal `just check` record gate also runs on macOS and requires no Java

| Tool | Install | Confirm |
|---|---|---|
| Go | Follow [Go installation](https://go.dev/doc/install) for the version in `go.mod` | `go version` |
| Git | Follow [Git installation](https://git-scm.com/downloads) | `git --version` |
| just | Follow [just installation](https://just.systems/man/en/packages.html) | `just --version` |
| mise | Follow [mise installation](https://mise.jdx.dev/getting-started.html) | `mise --version` |

`verification/publication/tools.json` owns the Java version, TLC download URL, and jar SHA-256
`just verify-publication` acquires those tools without changing the repository's Go dependencies
Network access is required for missing Go modules and the pinned tools
Set `TMPDIR` to a disk with space for the source snapshot, TLC states, and private probe files
The runner preserves detailed logs there
It writes the committed record only after every required check succeeds
