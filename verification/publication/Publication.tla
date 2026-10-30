---------------------------- MODULE Publication ----------------------------
EXTENDS Naturals, FiniteSets, TLC

CONSTANTS Bug, AllowKill, AllowCleanupRefusal
Jobs == {1, 2}
New(i) == IF i = 1 THEN "new1" ELSE "new2"
Partial(i) == IF i = 1 THEN "partial1" ELSE "partial2"
PublicComplete == {"absent", "old", "new1", "new2"}

VARIABLES pc, stage, force, signal, verdict, committed, public,
          input, unauthorized, cleanupRefused
vars == <<pc, stage, force, signal, verdict, committed, public,
          input, unauthorized, cleanupRefused>>

Init ==
  /\ pc = [i \in Jobs |-> "mkdir"]
  /\ stage = [i \in Jobs |-> "none"]
  /\ force \in [Jobs -> BOOLEAN]
  /\ signal = [i \in Jobs |-> 0]
  /\ verdict = [i \in Jobs |-> 999]
  (* 999 means no commit; otherwise record the signal present at commit. *)
  /\ committed = [i \in Jobs |-> 999]
  /\ public \in {"absent", "old"}
  /\ input = "original"
  /\ unauthorized = FALSE
  /\ cleanupRefused = [i \in Jobs |-> FALSE]

(* V = ncly-video-convert at 752ff5c8; N = Nicely SDK at v0.23.4.
   Model program counters and ghost observations are not extra code effects.
   Syscalls are separate steps. No action combines a context check and commit.
   Jobs share the destination and preserved input, with private staging dirs.
   Both jobs enter after preparation; destination is checked again in publish.
   Worker writes are abstract final byte writes, not a claim about video decode. *)

(* V/convert.go:283, one successful os.MkdirTemp. *)
Mkdir(i) ==
  /\ pc[i] = "mkdir"
  /\ pc' = [pc EXCEPT ![i] = "write"]
  /\ stage' = [stage EXCEPT ![i] = "empty"]
  /\ UNCHANGED <<force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:330-331 invokes worker; external worker's first staged write.
   Bug=partial represents changing that write's path to the public destination.
   Bug=input represents changing that write's path to the input. *)
WorkerPartial(i) ==
  /\ pc[i] = "write" /\ stage[i] = "empty"
  /\ stage' = [stage EXCEPT ![i] = "partial"]
  /\ public' = IF Bug = "partial" THEN Partial(i) ELSE public
  /\ input' = IF Bug = "input" THEN "damaged" ELSE input
  /\ UNCHANGED <<pc, force, signal, verdict, committed, unauthorized,
                 cleanupRefused>>

(* V/convert.go:331-339, worker's final write before successful return.
   Worker success is a separate action, to allow signal/failure between them. *)
WorkerComplete(i) ==
  /\ pc[i] = "write" /\ stage[i] = "partial"
  /\ stage' = [stage EXCEPT ![i] = "complete"]
  /\ UNCHANGED <<pc, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:331-339, successful program return observed by handBrake. *)
WorkerSuccess(i) ==
  /\ pc[i] = "write" /\ stage[i] = "complete"
  /\ pc' = [pc EXCEPT ![i] = "ctx"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:295, final ctx.Err read on no-normalize path. *)
CheckContext(i) ==
  /\ pc[i] = "ctx"
  /\ pc' = [pc EXCEPT ![i] = IF signal[i] = 0 THEN "stageStat" ELSE "cleanup"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:448-451, successful staged os.Lstat. The stage is private
   and the worker has returned, so its completeness remains stable. *)
StatStage(i) ==
  /\ pc[i] = "stageStat"
  /\ pc' = [pc EXCEPT ![i] = "destStat"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:453 and 507-523, destination os.Lstat snapshot. A second
   publisher can change the destination after this observation. *)
CheckDestination(i) ==
  /\ pc[i] = "destStat"
  /\ pc' = [pc EXCEPT ![i] =
       IF public = "absent" \/ force[i] THEN "commit" ELSE "cleanup"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:457, one successful rename when forced; 461, one hard link
   when not forced. Bug=replace changes the latter syscall to rename. *)
Commit(i) ==
  /\ pc[i] = "commit"
  /\ force[i] \/ public = "absent" \/ Bug = "replace"
  /\ public' = New(i)
  /\ unauthorized' = (unauthorized \/ (~force[i] /\ public # "absent"))
  /\ committed' = [committed EXCEPT ![i] = signal[i]]
  /\ pc' = [pc EXCEPT ![i] = "postStat"]
  /\ stage' = [stage EXCEPT ![i] = IF force[i] THEN "empty" ELSE @]
  /\ UNCHANGED <<force, signal, verdict, input, cleanupRefused>>

(* V/convert.go:461-465, os.Link atomically refuses the now-existing path. *)
LinkExists(i) ==
  /\ pc[i] = "commit" /\ ~force[i] /\ public # "absent" /\ Bug # "replace"
  /\ pc' = [pc EXCEPT ![i] = "cleanup"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:469-475, postcommit os.Stat. Even a failed stat cannot undo
   publication. This action permits either post-stat outcome via Failure. *)
StatPublished(i) ==
  /\ pc[i] = "postStat"
  /\ pc' = [pc EXCEPT ![i] = "cleanup"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:283-285, 295, 332-334, 448-451, 466-471. A failed syscall
   or worker return transfers control to deferred cleanup, never rollback. *)
Failure(i) ==
  /\ pc[i] \in {"mkdir", "write", "stageStat", "destStat", "commit", "postStat"}
  /\ pc' = [pc EXCEPT ![i] = "cleanup"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:288, successful RemoveAll's last removal or no-op.
   Earlier removals leave stage present and refine to stuttering. *)
Cleanup(i) ==
  /\ pc[i] = "cleanup"
  /\ stage' = [stage EXCEPT ![i] = "none"]
  /\ pc' = [pc EXCEPT ![i] = "verdict"]
  /\ UNCHANGED <<force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/convert.go:288-291, RemoveAll failure retains staging. *)
CleanupRefusal(i) ==
  /\ AllowCleanupRefusal /\ pc[i] = "cleanup" /\ stage[i] # "none"
  /\ pc' = [pc EXCEPT ![i] = "verdict"]
  /\ cleanupRefused' = [cleanupRefused EXCEPT ![i] = TRUE]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized>>

(* N/sdk/runner/signal.go:51-52, first caught signal cancels the context.
   Signals after the SDK's verdict read are outside this process's verdict. *)
CatchSignal(i, s) ==
  /\ signal[i] = 0 /\ pc[i] \notin {"done", "killed"}
  /\ signal' = [signal EXCEPT ![i] = s]
  /\ UNCHANGED <<pc, stage, force, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

(* N/sdk/extension/handle.go:70-71, ctx.Err sample for signal verdict;
   contract/answer.go:102-106 maps that signal to 130/143. Ordinary error
   details do not affect the invariants and are represented by verdict 0. *)
Decide(i) ==
  /\ pc[i] = "verdict"
  /\ verdict' = [verdict EXCEPT ![i] = signal[i]]
  /\ pc' = [pc EXCEPT ![i] = "done"]
  /\ UNCHANGED <<stage, force, signal, committed, public, input,
                 unauthorized, cleanupRefused>>

(* V/spec.md, Limits: SIGKILL prevents deferred cleanup. N/sdk/extension/handle.go:
   57-58 also documents host killing an extension after its cleanup budget. *)
HardKill(i) ==
  /\ AllowKill /\ pc[i] \notin {"done", "killed"}
  /\ pc' = [pc EXCEPT ![i] = "killed"]
  /\ UNCHANGED <<stage, force, signal, verdict, committed, public, input,
                 unauthorized, cleanupRefused>>

JobStep(i) == Mkdir(i) \/ WorkerPartial(i) \/ WorkerComplete(i)
           \/ WorkerSuccess(i) \/ CheckContext(i) \/ StatStage(i)
           \/ CheckDestination(i) \/ Commit(i) \/ LinkExists(i)
           \/ StatPublished(i) \/ Failure(i) \/ Cleanup(i)
           \/ CleanupRefusal(i) \/ Decide(i)
Next == \E i \in Jobs : JobStep(i) \/ HardKill(i)
                       \/ (\E s \in {130, 143} : CatchSignal(i, s))
Spec == Init /\ [][Next]_vars /\ (\A i \in Jobs : WF_vars(JobStep(i)))

CompletePublic == public \in PublicComplete
InputPreserved == input = "original"
Consent == ~unauthorized
SignalMeansNoCommit == \A i \in Jobs : verdict[i] \in {130, 143} => committed[i] = 999
NoCanceledCommit == \A i \in Jobs : committed[i] \notin {130, 143}
SignalAfterCommitOnly == \A i \in Jobs : signal[i] = 0 \/ committed[i] # 999
NoStageAfterExit == \A i \in Jobs : pc[i] \in {"done", "killed"} => stage[i] = "none"
EventuallyClean == <> (\A i \in Jobs : stage[i] = "none" /\ pc[i] = "done")
=============================================================================
