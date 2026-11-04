# Coverage record

Correspondence was confirmed against ncly-video-convert `7c17b027e8287503368c49f5c8e2730941095ede` and Nicely SDK `v0.26.0`
Advance these references only after reviewing the mapped operations
`verification/publication/record.json` binds the exact source bytes and tool pins from the completed run
`internal/publication/record.go` defines the complete input inventory

| Source | Functions and boundary |
|---|---|
| `convert.go:286-313` | `job.encode`, private staging, context sampling, deferred cleanup |
| `convert.go:332-345` | `job.handBrake`, worker return and staged file validation |
| `convert.go:451-481` | `job.publish`, staged stat, destination check, atomic link or rename, published stat |
| `convert.go:511-527` | `checkDestination`, alias and type refusal, overwrite consent |
| SDK `sdk/runner/signal.go:45-59` | `ListenForSignals`, first signal cancels context |
| SDK `sdk/extension/handle.go:48-81` | `loaded.handle`, handler completion, cleanup, cancellation sample |
| SDK `sdk/contract/answer.go:101-107` | `Registry.Decide`, signal precedence |
| SDK `sdk/command/answer.go:75-101` | `Report`, signal answer retains result data |

The model has two conforming publishers, eight initial states, and one conversion per publisher
They share a destination and preserved input, with separate private staging directories
The model explores all force combinations and initially absent or complete existing output
It admits failures and each publisher's first SIGINT or SIGTERM
Each atomic publication action corresponds to one `os.Link` or `os.Rename`
Context sampling and publication are separate actions

Safety assumes POSIX atomic publication, stable nonhostile paths, quiescent workers, and successful worker return for complete staged bytes
Eventual cleanup also requires fair handler progress, returning workers, and a cooperating filesystem
The executable probe uses `--no-normalize` and observes real Linux rename events before signaling
Extra private files lengthen actual deferred cleanup without a production hook

The pass covers this bounded design, rather than a full refinement proof of Go
It excludes normalization and GPU fallback internals, independent media decoding, hostile alias changes, and competing-writer result metadata
It also excludes macOS execution, host signal forwarding, power-loss durability, and fsync guarantees
SIGKILL, exceeded cleanup budgets, and filesystem refusal can leave staging
Post-publication errors can accompany an output that already exists

`verification/tlc.json` declares the model configurations and tool pins used by `sdk/verification`
`verification/tlc-record.json` holds the model outcomes
`verification/publication/record.json` holds the executable probe, mutation and restoration outcomes and binds the adapter, SDK signal implementation, probe and runner
The three executable results remain explicit as readable publication evidence for [nicely#427](https://github.com/pascalandy/nicely/issues/427)
The mutation result must show its expected failure, between the passing probe and restoration
The SDK omits nondeterministic counterexample graph counts; the Safety exploration records its complete graph
Run `just verify-publication` and commit both records together
