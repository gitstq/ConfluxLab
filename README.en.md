# ConfluxLab 🧬

> **A zero-dependency CRDT toolkit and a randomized concurrency sandbox.**
> Turning "eventual consistency" from a claim into a regression test you can run in CI.

[简体中文](README.md) · [繁體中文](README.zh-TW.md) · **English**

[![CI](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml/badge.svg)](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-%E2%89%A5%201.21-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Dependencies](https://img.shields.io/badge/runtime%20dependencies-0-brightgreen)](#-features)

> 📦 **Prefer not to install Go?** Grab the [**v1.0.0 release**](https://github.com/gitstq/ConfluxLab/releases/tag/v1.0.0) — five static binaries for Linux / macOS / Windows, with `SHA256SUMS.txt` checksums. No setup required.

---

## 🎉 About the project

"We use CRDTs, so replicas converge automatically." Everyone says it. But **how do you actually prove convergence** — by rereading the paper, or by waiting for a production incident?

ConfluxLab takes a different approach: **make convergence something you can execute, measure, and replay**.

It is a library built entirely on the Go standard library, shipping five state-based CRDTs plus a vector clock. More importantly, it ships a **deterministic randomized concurrency sandbox** that interleaves local writes, bidirectional sync, network partitions, blocked syncs, and duplicate message delivery across several replicas. Once the schedule finishes, partitions heal automatically, every replica's snapshot is compared, and the run also checks monotonicity, tombstone integrity, and the three algebraic laws. Break any of them and the process exits non-zero, failing CI immediately.

| Question | ConfluxLab's answer |
| --- | --- |
| Does out-of-order delivery break anything? | Commutativity gate: `a⊔b = b⊔a` |
| Are different merge topologies equivalent? | Associativity gate: `(a⊔b)⊔c = a⊔(b⊔c)` |
| Is message retry safe? | Idempotence gate: `a⊔a = a`, plus real duplicate delivery in the sandbox |
| Do replicas agree after a partition heals? | Snapshot comparison after automatic healing |
| How do I reproduce a flaky failure? | Pass `-seed` and replay the entire schedule bit-for-bit |

**The problem it solves.** CRDT bugs tend to surface only under one particular interleaving of concurrency, partition, and retransmission. A handful of hand-written unit tests will never find them. ConfluxLab replaces manual enumeration with hundreds of randomized schedules — and every failure is **exactly reproducible**.

**What makes it different.** Most projects in this space are "yet another CRDT implementation." ConfluxLab puts the weight on the **verifier**: algebraic-law checks, invariant tracking, deterministic replay, machine-readable reports, and CI gates — all written from scratch, with no external dependencies.

**Inspiration.** Drawn from the 2026-11-08 GitHub Trending list: `cloudflare/security-audit-skill` (security conclusions as independently verifiable, machine-readable findings) and `mbillow/localthings` (local-first, no cloud account required). The former inspired verifiable results; the latter inspired the zero-dependency, offline-capable design. All code, algorithms, and documentation here are independently written; no reference implementation was copied.

---

## ✨ Features

- 🧬 **Five state-based CRDTs, all written in-house**: `GCounter`, `PNCounter`, `LWWRegister`, `MVRegister`, and `ORSet`.
- 🕰️ **Vector clocks** with full `before / after / equal / concurrent` causality detection, powering concurrent-version tracking in `MVRegister`.
- ⚖️ **Executable algebraic-law checks**: randomly generated state triples are checked for commutativity, associativity, and idempotence — the defining properties of a join-semilattice and a sufficient condition for convergence.
- 🌐 **Randomized concurrency sandbox**: local writes, bidirectional merges, partition flips, blocked syncs, and duplicate delivery are interleaved, followed by automatic healing and per-replica comparison.
- 🎯 **Deterministic replay**: a built-in SplitMix64 generator, no global `math/rand` source, and no `time.Now()` in scheduling decisions. **The same seed always produces a bit-identical run** — diff two `-json` outputs to confirm.
- 🛡️ **Conflicts stay visible**: `LWWRegister` picks a winner through a total order of `(timestamp, node)`; `MVRegister` keeps every concurrent version for the caller to resolve. If two versions share an identifier but disagree on the value, `Merge` returns an error instead of silently choosing one.
- 📊 **Machine-readable reports as a gate**: `-json` emits a structured report with op counts, merges, blocked syncs, partition flips, violations, and final snapshots per type. Any anomaly yields exit code 1.
- 🪶 **Zero third-party dependencies**: runtime and tests use only the Go standard library — `go mod download` has nothing to do.
- 📦 **Five static binary targets**: Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64), built with `CGO_ENABLED=0` and shipped with SHA256 checksums.

---

## 🚀 Quick start

**Requirements**: Go ≥ 1.21 (or just grab a prebuilt binary). Nothing else to install.

```bash
# Option 1 — run from source, no dependency installation
git clone https://github.com/gitstq/ConfluxLab.git
cd ConfluxLab
go run ./cmd/confluxlab verify

# Option 2 — build a single static binary
make build
./dist/confluxlab version

# Option 3 — use it as a library
go get github.com/gitstq/ConfluxLab
```

The three commands you'll reach for most:

```bash
# 1) Randomized sandbox: prove replicas converge
go run ./cmd/confluxlab verify -rounds 500 -replicas 5

# 2) Algebraic-law gate: commutativity / associativity / idempotence
go run ./cmd/confluxlab laws -rounds 300

# 3) Three concrete concurrency semantics, demonstrated
go run ./cmd/confluxlab demo
```

Typical output from a first `verify` run:

```text
ConfluxLab · Randomized convergence verification
──────────────────────────────────────────────────────────────────────────────
  seed=20261108 rounds=300 replicas=4 localOps=661 merges=373 kinds=5 status=CONVERGED
──────────────────────────────────────────────────────────────────────────────
  type            converged   localOps   merges  dupes  blocked  parts  violations
  gcounter       ✅ yes           138       66        6       60       36      0
  pncounter      ✅ yes           141       71        7       54       34      0
  lwwregister    ✅ yes           131       77       17       58       34      0
  mvregister     ✅ yes           119       80       15       52       49      0
  orset          ✅ yes           132       79       14       60       29      0
──────────────────────────────────────────────────────────────────────────────
  Result: all replicas converged; all three laws and invariants passed ✅
```

---

## 📖 Usage guide

### `verify` — randomized convergence checks

| Flag | Default | Meaning |
| --- | --- | --- |
| `-seed` | `20261108` | RNG seed; **same seed ⇒ bit-identical run** |
| `-rounds` | `400` | Scheduling events per CRDT type |
| `-replicas` | `4` | Number of replicas (≥ 2) |
| `-kinds` | `all` | Comma-separated types, e.g. `orset,mvregister` |
| `-json` | `false` | Emit a machine-readable JSON report |
| `-timeline` | `false` | Print the event timeline (partitions, blocked syncs, duplicates) |

```bash
# Run two types only, with the partition timeline
go run ./cmd/confluxlab verify -seed 42 -rounds 120 -kinds orset,mvregister -timeline

# JSON for scripting, or to assert deterministic replay
go run ./cmd/confluxlab verify -rounds 200 -json > run-a.json
go run ./cmd/confluxlab verify -rounds 200 -json > run-b.json
diff run-a.json run-b.json   # identical for the same seed
```

Timeline excerpt:

```text
  [orset]  converged=true  localOps=58  merges=20  dupes=2  blocked=27  partitionFlips=15
  │ #001  partition flip  n1 → partition 1
  │ #002  sync blocked    n1 ↔ n2  (partition 1|0)
  │ #003  bidirectional sync  n2 ↔ n3
  │ #006  local write   remove(epsilon)@n1#1         → n1
  │ #010  local write   add(beta)@n1#2               → n1
  │ #011  local write   remove(beta)@n0#3            → n0
  │ ...   duplicate delivery n2 ↔ n3  (idempotence check)
  ──── network heals: fully-connected anti-entropy ────
  │ converged=true  finalSnapshot={[alpha beta delta epsilon gamma] 38}
```

> 💡 **Why blocked syncs matter.** Merging only in a fully connected network barely tests anything. Partitions let replicas evolve into genuinely concurrent versions; healing is where convergence is actually examined.

### `laws` — the algebraic-law gate

```bash
go run ./cmd/confluxlab laws -seed 20261108 -rounds 300
```

```text
  type                     checks    violations   result
  gcounter                900          0          ✅ pass
  pncounter               900          0          ✅ pass
  lwwregister             900          0          ✅ pass
  mvregister              900          0          ✅ pass
  orset                   900          0          ✅ pass
  ────────────────────────────────────────────────────────────────
  commutativity a⊔b = b⊔a | associativity (a⊔b)⊔c = a⊔(b⊔c) | idempotence a⊔a = a
```

### `demo` — three concrete semantics

```text
  ① ORSet add-wins: one side removes, the other adds the same element during a partition
     visible elements after merge = [task]  → concurrent add survives (its tag was unobserved)
  ② LWWRegister tie-break: two concurrent writes with identical timestamps
     both replicas independently converge on "來自 n1" (node-name lexicographic order)
  ③ MVRegister exposes conflicts instead of silently dropping them
     concurrent versions = [配置 A 配置 B]  → resolved by the caller, not silently overwritten
```

### As a library

```go
package main

import (
    "fmt"
    "github.com/gitstq/ConfluxLab"
)

func main() {
    left, right := confluxlab.NewORSet(), confluxlab.NewORSet()
    left.Add("deploy", "n0:1")
    _ = right.Merge(left.Clone())
    left.Remove("deploy")       // n0 observes tag n0:1
    right.Add("deploy", "n1:9") // concurrent add during a partition, tag unobserved

    _ = left.Merge(right.Clone())
    _ = right.Merge(left.Clone())
    fmt.Println(left.Elements()) // [deploy] — add-wins
}
```

A runnable version lives in [`examples/basic/main.go`](examples/basic/main.go).

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | All types converged; all laws and invariants passed |
| `1` | Non-convergence, invariant violation, or runtime error |
| `2` | Invalid arguments / usage error |

So it drops straight into a script or pipeline:

```bash
go run ./cmd/confluxlab verify -rounds 500 -replicas 5 || exit 1
```

### FAQ

**Different seeds give different results — isn't that flaky?**
Quite the opposite. The seed determines the entire schedule, and the same seed is bit-identical. Multiple seeds simply widen coverage; CI runs both seeded schedules and conventional unit tests.

**`MVRegister` returns several values. Bug?**
No — that's deliberate. It **refuses to discard** concurrent writes and hands the conflict to the caller. If you want an automatic winner, use `LWWRegister`.

**Will the verifier cover a CRDT I add myself?**
Yes. Register it in `AllKinds()` following the checklist in [`CONTRIBUTING.md`](CONTRIBUTING.md), and both `laws` and `verify` pick it up automatically.

---

## 💡 Design and roadmap

### Why the three laws are the foundation of convergence

`Merge` on a state-based CRDT is a **least upper bound (join)** over a partially ordered set. If it satisfies:

| Law | Meaning | Real-world counterpart |
| --- | --- | --- |
| Commutativity `a⊔b = b⊔a` | Result is independent of merge order | Out-of-order message arrival |
| Associativity `(a⊔b)⊔c = a⊔(b⊔c)` | Result is independent of merge topology | Star / gossip / chain sync |
| Idempotence `a⊔a = a` | Re-merging has no effect | Retransmission, at-least-once delivery |

together they form a **join-semilattice**, and replicas converge without any coordination. ConfluxLab doesn't leave these as a comment — it turns them into randomized regression tests that run on every CI build.

### The sandbox's event model

```text
N replica states + N vector clocks + N partition groups
        ↓ each round picks one event at random
  ├─ local write (45%)  → advance local Lamport stamp & vector clock, apply Op
  ├─ bidirectional sync (43%)  → allowed within a partition; blocked across; may be duplicated
  └─ partition flip (12%)  → move one replica to the other partition
        ↓ schedule ends
  healing: drop all partitions, repeat pairwise anti-entropy until stable
        ↓
  per-replica snapshot comparison + monotonicity / tombstone checks + structured report
```

### Key decisions

1. **Determinism beats "more randomness".** All randomness comes from `RNG`; `time.Now()` never influences scheduling. A single seed is enough to replay a failure in full — invaluable when debugging convergence.
2. **Globally unique write identifiers are a hard precondition.** `(timestamp, node)` in `LWWRegister` and tags in `ORSet` must never collide. During development the law checker caught a false commutativity failure caused precisely by a generator violating this assumption; `Merge` now reports such collisions explicitly (see [`CHANGELOG.md`](CHANGELOG.md)).
3. **Conflicts stay visible.** `Merge` must never silently lose information: either a total order picks a winner, or every concurrent version is retained.
4. **Zero dependencies.** No CLI framework, no test framework, no assertion library. Lower supply-chain risk, and `go test ./...` works anywhere.

### Roadmap

1. Sequence CRDTs (RGA / LSEQ) for collaborative text, plus tombstone compaction
2. Operation-based delta transfer with bandwidth accounting, to quantify full-state anti-entropy
3. Automatic shrinking to a minimal reproducing schedule (delta debugging)
4. Byzantine replica injection — simulating a node that sends malformed state
5. JSON Schema export so editors can autocomplete report fields
6. End-to-end validation against a real transport (WebSocket / NATS)

> The full derivation is in [`docs/design.md`](docs/design.md).

---

## 📦 Packaging and deployment

ConfluxLab is **a library plus a CLI**, not a desktop app with an installer. It ships in two forms.

### As a library

```bash
go get github.com/gitstq/ConfluxLab
```

No external dependencies, no configuration. Module path: `github.com/gitstq/ConfluxLab`.

### Static binaries for every platform

```bash
make cross                  # equivalent to bash scripts/build.sh
# or pin the version
VERSION=v1.0.0 bash scripts/build.sh
```

Output in `dist/`:

| Target | Artifact |
| --- | --- |
| Linux amd64 / arm64 | `confluxlab-v1.0.0-linux-amd64` / `...-linux-arm64` |
| macOS amd64 / arm64 | `confluxlab-v1.0.0-darwin-amd64` / `...-darwin-arm64` |
| Windows amd64 | `confluxlab-v1.0.0-windows-amd64.exe` |
| Checksums | `SHA256SUMS.txt` |

```bash
sha256sum -c dist/SHA256SUMS.txt

# Run directly (Linux example; statically linked, no runtime deps)
./dist/confluxlab-v1.0.0-linux-amd64 verify -rounds 500
```

> ⚠️ The macOS and Windows artifacts are cross-compiled on Linux. Build success and file formats are verified, but runtime behavior has not been exercised on those platforms — run a `version` and `verify` smoke test on the target machine.

### CI integration

The repo includes [`.github/workflows/ci.yml`](.github/workflows/ci.yml): a Go 1.21/1.22/1.23 matrix running formatting checks, `go vet`, `go test -race`, build, the law gate, a 5-replica / 500-event convergence gate, a deterministic-replay diff, and a **hardcoded-credential scan**. Any failure blocks the merge.

Outside GitHub, the minimal setup is two commands:

```bash
go test ./... -count=1
go run ./cmd/confluxlab verify -rounds 500 -replicas 5
```

---

## 🤝 Contributing

New CRDT types, stronger invariants, and further translations are all welcome. Please read [`CONTRIBUTING.md`](CONTRIBUTING.md) first — the essentials:

- Commit messages follow the **Angular convention**: `feat:` / `fix:` / `docs:` / `refactor:` / `test:`
- A new type must implement `State`, register in `AllKinds()`, come with unit tests, and pass `make all`
- Three red lines: **no runtime dependencies**, **determinism first**, **conflicts stay visible**
- When filing an issue, include `-seed / -rounds / -replicas` and the `-json` output so the failure can be reproduced exactly

---

## 📄 License

Released under the [MIT License](LICENSE).

- ✅ Commercial use, modification, distribution, and sublicensing are permitted
- ✅ No fees, no additional restrictions
- ⚠️ The copyright and license notices must be retained
- ⚠️ Provided "as is", without warranty of any kind

Full terms in [`LICENSE`](LICENSE). Copyright (c) 2026 gitstq.
