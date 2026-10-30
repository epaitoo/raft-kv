# Design Decisions

Raft Key-Value Store. Decisions made before Phase 0, October 2026.

---

## Decision 1: Log indexing

### Context
Figure 2 uses a 1-indexed log, but Go slices are 0-indexed.

### Options
- **Option A**: Special-case index 0 everywhere it is used.
- **Option B**: Put a sentinel entry at slice position 0.

### Choice
Option B. Real entries start at slice position 1, which is log index 1. A sentinel entry sits at index 0 with term 0.

### Why Option B beats Option A
With option A, every place that looks at the start of the log needs its own special case for index 0. We already found two: the AppendEntries consistency check when `prevLogIndex` is 0, and the last log term of an empty log in the RequestVote comparison. Each special case is a branch I have to remember to write, and forgetting one is a bug that only appears in a brand new node in the cluster with no entries.

With option B, the sentinel means every check that looks at index 0 finds an entry with term 0, which every node's log shares, so the same code runs whether the log is empty or not. One rule everywhere, fewer places for off-by-one bugs.

### Why it's safe
- `currentTerm` is initialized to 0.
- It increments at the start of an election, so any leader has a term of at least 1.
- A leader stamps each new entry with the term in which it received the entry.

Therefore no real entry can ever have term 0, so the sentinel can never be mistaken for one.

### Cost
`len(log)` does not mean the number of real entries. The number of real entries, and the last log index, is `len(log) - 1`. No empty check is needed.

### Revisit if
- **Restart**: Save only the real entries to disk. On every startup, create the sentinel first, then append the loaded entries. This also covers the very first boot, when there is no file yet.
- **Snapshotting**: Snapshotting would discard the start of the log, so the position equals index rule would no longer hold. To support it, I would store `lastIncludedIndex` and convert with:
  `slice position = log index - lastIncludedIndex`

---

## Decision 2: Election timer

### Context
Followers need an election timeout that only valid events reset.

### Options
- **Option A**: `time.Timer` plus a last reset time check.
- **Option B**: A loop that wakes every 10ms plus a last reset time check.

### Choice
Option A.

There are handler goroutines and one timer goroutine.
- **Handler goroutines**: The AppendEntries handler (on a valid heartbeat from the current leader) and the RequestVote handler (when it grants a vote) write the last reset time. They never touch the timer.
- **Timer goroutine**: The only goroutine that ever touches the timer. It checks the last reset time and either rearms the timer or starts an election.

```
Handler goroutines                     Timer goroutine
------------------                     ---------------
receives heartbeat,                    timer fires
  or grants a vote                     takes the lock
checks its term                        checks time since last reset
writes the last reset time             rearms, or starts an election
```

A new random timeout is picked from the interval each time a new waiting period starts, so repeated split votes are unlikely (section 5.2).

For the timer, I will go for local variable inside the goroutine because I dont want any handler in the code to see and even to be able to reset it .

### Why
Rearming is simple for me, and option A does not have to wake up every 10ms doing nothing.

### The rule

| Event | Resets? | Why |
|---|---|---|
| AppendEntries from the current leader | Yes | Proof the leader is alive |
| Granting a vote | Yes | Gives the candidate you backed time to win |
| Refusing a vote | No | Would silence a node that could win |
| Rejecting a stale AppendEntries | No | Would block elections while the real leader is dead |

Resetting on a refused vote or a rejected stale AppendEntries is a liveness failure: the deadline keeps moving away, no election starts, and the cluster never gets a working leader, even though nothing incorrect is ever committed.

### The race and how it's avoided
Only the timer goroutine ever touches the timer, so there is no `Reset` call from a handler to race against a timer that has just fired.

Before starting an election, the timer goroutine takes the lock and checks how long it has been since the last valid reset. Only if that is at least the timeout does it start an election. A fire that a heartbeat has already made stale is caught by this check.

### Cost
The timer must be rearmed with:
`remaining time = timeout - time since last reset`
This calculation is a place an arithmetic bug can hide.

### Revisit if
The rearm logic produces bugs I can't track down. Then switch to option B, which reuses the same check.

---

## Decision 3: Transport

### Context
Nodes must send RequestVote and AppendEntries to peers, and Week 9 tests must be able to partition, drop and delay those messages.

### Options
- **Option A**: Raft calls gRPC directly.
- **Option B**: Raft calls net/rpc directly.
- **Option C**: Raft talks to an interface I define. Tests use an in-process fake network; the real cluster uses net/rpc behind the interface.

### Choice
Option C.

### Why
Tests can partition, drop and delay messages without touching a real network. net/rpc because it is standard library, needs no code generation, and keeps the moving parts down while I'm learning.

### Interface contract
**AppendEntries**
- In: `term`, `leaderId`, `prevLogIndex`, `prevLogTerm`, `entries`, `leaderCommit`
- Out: `term`, `success`

**RequestVote**
- In: `term`, `candidateId`, `lastLogIndex`, `lastLogTerm`
- Out: `term`, `voteGranted`

**Failure**: If the peer is dead or unreachable, the sender gets an error (a timeout or a refused connection). Raft must never treat an error as a refusal; it just means "no information, try again later".

**Who decides how long to wait**: Raft code, because only Raft knows its timing: the heartbeat interval and election timeout decide how long is too long.

### Cost
net/rpc has no built-in timeout, so I build it myself, and the package is frozen.

### Revisit if
gRPC is needed later. Then add a second implementation of the interface; Raft doesn't change.

---

## Decision 4: Concurrency model

### Context
Several goroutines touch the same node state: RPC handlers, the election timer goroutine, and the leader's per-peer senders all read and write the term, role, log and last reset time.

### Options
- **Option A**: One mutex plus goroutines.
- **Option B**: A single event loop with channels.

### Choice
Option A, with a single mutex per node.

### Why
- **Consistency**: Decision 2 already has handlers and a timer goroutine sharing state under a lock.
- **Simplicity**: It is simple for me to reason about.
- **Diagnosis**: A mistake is usually a data race, which `-race` finds for me.
- **One mutex**: There is never a question of which lock guards which field, and no risk of two locks being taken in different orders and deadlocking.

### Rules
1. Never hold the lock while making an RPC or waiting on anything slow; release it, make the call, then lock again.
2. After a reply returns and I lock again, check that my term and role are still what they were when I sent the request; if either changed, discard the reply.

### Cost
Handlers never run in parallel, which is fine at five nodes. Every reply handler must follow Rule 2, so one forgotten check is a bug.

### Revisit if
Profiling shows lock contention slowing the cluster, or a goroutine is found holding the lock across slow work that can't be avoided.
