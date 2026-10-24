# AGENTS.md

`ncly-video-convert` adds commands at `ncly video convert` and beneath it.
The domain names a family of work, such as `video`.
An optional name identifies one job, such as `convert`.
Neither names the wrapped tool.
Read `extension-identity.md` in the pinned contract for the slug and command ownership.

## Read, in this order

1. `vision.md`: the job of the extension, what it wraps, and what it refuses
2. `spec.md`: what proves the success of each command, why each effect exists, and its failures
3. `manifest.toml`: the machine facts of each command, which `ncly describe video convert --json` shows once the extension is installed
4. The contract, at the Nicely version that `go.mod` pins. `just contract` prints its folder. Start with "What a great extension does" in its `extensions.md`, and open only the parts that it links

## Change a command

1. `spec.md`: its section, titled `## ncly <path>`, at the extension root or beneath it
2. `manifest.toml`: its declaration, with its effects, programs, keys, and result schema
3. `locales/en.toml`: its text, each entry with a description
4. Add a placeholder handler that compiles, and register it in `extension.go`
5. `testdata/script/`: its scenarios, which fail for the missing behavior. `test-kit.md` in the contract lists their commands
6. Write the handler body, on `sdk/wrap` for one program, or on `sdk/runner` for more
7. `skill/SKILL.md`: when to use the command, and when not to

`extension.Check` must pass before a scenario can fail for the missing behavior. Steps 1 through 4 prepare that check.

A wrapped script goes in `worker/`. To take in a script, or to review the extension when the user asks, use the skill in `.agents/skills/nicely-extension/`.

## Recipes

The recipes need Go, `just`, and `uv`, which runs the pinned Copier of `just update`. Scenarios use a shell stub of HandBrakeCLI. A real conversion needs HandBrakeCLI on PATH.

After Copier creates the repository, run `go mod tidy` once, which writes `go.sum` for the recipes. Then commit the repository with Git, since `just update` needs a clean Git tree. Run `lefthook install` once per clone, so `just check` runs before each push.

- `just check`: every check. It runs `go mod tidy -diff`, `gofmt`, `go vet`, and `go test`, whose test kit installs the extension into scratch folders
- `just install`: installs the extension into Nicely's extensions folder, in place of its previous install
- `just update <version>`: moves the repository to another Nicely version, the template first and then the SDK, on a clean Git tree
- `just contract`: prints the folder of the contract at the Nicely version that `go.mod` pins

## Done when

`just check` passes, and the installed command ran once on real input. For a paid command, its dry run is enough. For a command that needs a key, relay the grant hint to the human and wait.
