# Interpret the results

Exit 0 from the runner means every check produced its required result
It includes expected counterexamples and broken variants
It does not claim that cancellation prevents publication

| Check | Required result |
|---|---|
| `Safety.cfg` | Complete exploration passes completeness, input preservation, consent, terminal cleanup, and eventual cleanup |
| `Partial.cfg` | Exit 12, `CompletePublic` violation after a misdirected worker write |
| `Replace.cfg` | Exit 12, `Consent` violation when a second nonforced rename replaces the first result |
| `Input.cfg` | Exit 12, `InputPreserved` violation after a misdirected input write |
| `Signal.cfg` | Exit 12, `SignalMeansNoCommit` violation because a signal verdict can follow publication |
| `SignalBeforeCommit.cfg` | Exit 12, `NoCanceledCommit` violation because cancellation can follow the final context check |
| `SignalAfterCommit.cfg` | Exit 12, `SignalMeansNoCommit` violation restricted to signals after publication |
| `Kill.cfg` | Exit 12, `NoStageAfterExit` violation because SIGKILL prevents cleanup |
| `CleanupRefusal.cfg` | Exit 12, `NoStageAfterExit` violation because the filesystem can refuse cleanup |
| `TestFVPublicationFailure` | Failed encode preserves input and previous output, with no remaining stage |
| `TestFVPublicationSignalAfterCommit` | Both signals return 130 or 143, preserve input, remove staging, and name the published output and size |
| Scratch mutation | Writing `partial\n` to the public destination fails the failure probe for the changed destination bytes |
| Restored probe | Both executable checks pass after scratch source restoration |

The signal counterexample follows this order:

1. HandBrake finishes its private staged result
2. `encode` samples its context and continues
3. `publish` commits the output through link or rename
4. The SDK receives SIGINT or SIGTERM during deferred cleanup
5. The handler returns its result data
6. The SDK returns 130 or 143 and retains the published output path and size

This is the existing contract working
The executable probe pins its agent-visible result
The shorter signal-before-commit ordering remains a model result without a deterministic executable reproduction

Logs remain in the scratch directory named by the runner
Keep them while interpreting traces
After review, delete that directory with `trash` and its literal absolute path
