# ConfluxLab 🧬

> **零依赖 CRDT 收敛性工具箱与随机并发验证沙盘**
> 把「最终一致性」从一句口号，变成可以跑进 CI 的回归测试。

**简体中文** · [繁體中文](README.zh-TW.md) · [English](README.en.md)

[![CI](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml/badge.svg)](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-%E2%89%A5%201.21-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Dependencies](https://img.shields.io/badge/runtime%20dependencies-0-brightgreen)](#-核心特性)

> 📦 **不想装 Go？** 直接下载 [**v1.0.0 Release**](https://github.com/gitstq/ConfluxLab/releases/tag/v1.0.0)：Linux / macOS / Windows 五平台静态二进制，附 `SHA256SUMS.txt` 校验和，开箱即用。

---

## 🎉 项目介绍

写分布式系统时，我们常说"用 CRDT 就能自动收敛"。但**怎么证明它真的收敛**？靠读论文，还是靠线上出一次事故？

ConfluxLab 的回答是：**把收敛性变成可执行、可测量、可回放的指标**。

它是一个纯 Go 标准库实现的工具箱，内置 5 种状态型 CRDT 与向量时钟；更关键的是附带一个**确定性的随机并发沙盘**——把多个副本的本地写入、双向同步、网络分区、同步受阻、消息重复投递交错执行，随后自动愈合分区，逐副本比对快照，并同时检查单调性、墓碑完整性，以及交换律 / 结合律 / 幂等律。任何一条被破坏，进程返回非零退出码，CI 直接失败。

| 你要回答的问题 | ConfluxLab 给出的答案 |
| --- | --- |
| 消息乱序会不会出错？ | 交换律门禁：`a⊔b = b⊔a` |
| 不同的合并拓扑等价吗？ | 结合律门禁：`(a⊔b)⊔c = a⊔(b⊔c)` |
| 消息重传安全吗？ | 幂等律门禁：`a⊔a = a`，并在沙盘中真实重复投递 |
| 分区恢复后副本一致吗？ | 沙盘愈合后逐副本比对快照 |
| 线上偶发问题怎么复现？ | 输入 `-seed` 即可逐位重放整条调度序列 |

**解决的痛点**：CRDT 实现的 bug 往往只在「并发 + 分区 + 重传」的特定交错下暴露，手写几个单元测试覆盖不到。ConfluxLab 用数百轮随机调度替代人工穷举，并且**失败可精确复现**。

**自研差异化亮点**：市面上多数项目是"再实现一遍 CRDT"，本项目把重心放在**验证器**上——代数律检查、不变量追踪、确定性回放、机器可读报告、CI 门禁，全部自研且零依赖。

**灵感来源**：2026-11-08 GitHub Trending 上的 `cloudflare/security-audit-skill`（把安全结论做成可独立验证、机器可读的发现）与 `mbillow/localthings`（本地优先、无云依赖）。前者启发了"让结论可验证"，后者启发了"零依赖、离线可跑"。本项目的代码、算法与文档均为独立自研，未复制任何参考项目实现。

---

## ✨ 核心特性

- 🧬 **5 种状态型 CRDT 全自研**：`GCounter`（只增计数器）、`PNCounter`（可增减计数器）、`LWWRegister`（最后写入者获胜）、`MVRegister`（多值寄存器）、`ORSet`（观察-删除集合）。
- 🕰️ **向量时钟**：完整判定 `before / after / equal / concurrent` 四种因果序，为 `MVRegister` 提供并发版本识别。
- ⚖️ **三条代数律可执行验证**：随机生成状态组，逐一检查交换律、结合律、幂等律——它们是 join-semilattice 的定义性质，也是收敛性的充分条件。
- 🌐 **随机并发沙盘**：本地写入、双向合并、分区切换、同步受阻、消息重复投递交错进行；自动愈合后逐副本比对。
- 🎯 **确定性可复现**：自研 SplitMix64 发生器，禁用 `math/rand` 全局源与 `time.Now()`；**同种子必得逐位一致的结果**，可用 `-json` 直接 diff 两次运行。
- 🛡️ **冲突显式化**：`LWWRegister` 用 `(时戳, 节点)` 全序确定胜者；`MVRegister` 保留所有并发版本交给上层；`Merge` 遇到同标识不同值直接报错，绝不静默吞掉。
- 📊 **机器可读报告 + 门禁**：`-json` 输出结构化报告，含每类型的操作数、合并数、受阻数、分区次数、违反项与最终快照；任一异常退出码为 1。
- 🪶 **零第三方依赖**：运行期与测试期均只用 Go 标准库，`go mod download` 无事可做。
- 📦 **5 目标静态二进制**：Linux (amd64/arm64)、macOS (amd64/arm64)、Windows (amd64)，`CGO_ENABLED=0`，附 SHA256 校验和。

---

## 🚀 快速开始

**环境要求**：Go ≥ 1.21（或直接使用预编译二进制）；无其他依赖。

```bash
# 方式一：源码运行，无需安装任何依赖
git clone https://github.com/gitstq/ConfluxLab.git
cd ConfluxLab
go run ./cmd/confluxlab verify

# 方式二：编译成单个静态二进制
make build
./dist/confluxlab version

# 方式三：作为库引入
go get github.com/gitstq/ConfluxLab
```

三条最常用的命令：

```bash
# ① 随机并发沙盘：验证多副本最终收敛
go run ./cmd/confluxlab verify -rounds 500 -replicas 5

# ② 代数律门禁：验证交换 / 结合 / 幂等
go run ./cmd/confluxlab laws -rounds 300

# ③ 三个真实并发语义演示（add-wins、LWW 平局、冲突暴露）
go run ./cmd/confluxlab demo
```

首次运行 `verify` 的典型输出：

```text
ConfluxLab · 随机并发收敛性验证
──────────────────────────────────────────────────────────────────────────────
  seed=20261108 rounds=300 replicas=4 localOps=661 merges=373 kinds=5 status=CONVERGED
──────────────────────────────────────────────────────────────────────────────
  类型             收敛           本地操作       合并       重复       受阻       分区     违反
  gcounter       ✅ 是           138       66        6       60       36      0
  pncounter      ✅ 是           141       71        7       54       34      0
  lwwregister    ✅ 是           131       77       17       58       34      0
  mvregister     ✅ 是           119       80       15       52       49      0
  orset          ✅ 是           132       79       14       60       29      0
──────────────────────────────────────────────────────────────────────────────
  结论: 全部收敛，三条代数律与全部不变量通过 ✅
```

---

## 📖 详细使用指南

### `verify` — 随机并发收敛性验证

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-seed` | `20261108` | 随机种子；**同种子必得逐位一致结果** |
| `-rounds` | `400` | 每种类型的调度事件数 |
| `-replicas` | `4` | 副本数量（≥ 2） |
| `-kinds` | `all` | 逗号分隔的类型，如 `orset,mvregister` |
| `-json` | `false` | 输出机器可读 JSON 报告 |
| `-timeline` | `false` | 输出事件时间线（含分区、受阻、重复投递） |

```bash
# 只跑两种类型，查看分区时间线
go run ./cmd/confluxlab verify -seed 42 -rounds 120 -kinds orset,mvregister -timeline

# 输出 JSON，供脚本消费或做确定性回放比对
go run ./cmd/confluxlab verify -rounds 200 -json > run-a.json
go run ./cmd/confluxlab verify -rounds 200 -json > run-b.json
diff run-a.json run-b.json   # 同种子应完全相同
```

时间线样例（节选）：

```text
  [orset]  收敛=true  本地操作=58  合并=20  重复=2  受阻=27  分区切换=15
  │ #001  分区切换  n1 → 分区 1
  │ #002  同步受阻  n1 ↔ n2  (分区 1|0)
  │ #003  双向同步  n2 ↔ n3
  │ #006  本地写入  remove(epsilon)@n1#1         → n1
  │ #010  本地写入  add(beta)@n1#2               → n1
  │ #011  本地写入  remove(beta)@n0#3            → n0
  │ ...  重复投递 n2 ↔ n3  (验证幂等)
  ──── 网络愈合：全连通反熵 ────
  │ 收敛=true  最终状态={[alpha beta delta epsilon gamma] 38}
```

> 💡 **为什么"受阻"很重要**：只做全连通下的合并几乎测不出问题。分区让不同副本独立演化出**真正并发**的版本，愈合阶段才是收敛性的考场。

### `laws` — 代数律门禁

```bash
go run ./cmd/confluxlab laws -seed 20261108 -rounds 300
```

```text
  类型                     检查次数         违反 结果
  gcounter                900          0 ✅ 通过
  pncounter               900          0 ✅ 通过
  lwwregister             900          0 ✅ 通过
  mvregister              900          0 ✅ 通过
  orset                   900          0 ✅ 通过
  ────────────────────────────────────────────────────────────────
  交换律 a⊔b = b⊔a ｜ 结合律 (a⊔b)⊔c = a⊔(b⊔c) ｜ 幂等律 a⊔a = a
```

### `demo` — 三个真实语义

```text
  ① ORSet add-wins：分区两侧分别对同一元素执行 remove 与 add
     合并后可见元素 = [task]  → 并发 add 存活（新 tag 未被观察到）
  ② LWWRegister 平局打破：相同时戳的两次并发写入
     两个副本独立合并后均得到 "来自 n1"（节点名字典序决定，结果确定）
  ③ MVRegister 不静默丢弃冲突：两个并发写入同时保留
     并发版本 = [配置 A 配置 B]  → 由上层决策，而非悄悄覆盖
```

### 作为库使用

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
    left.Remove("deploy")       // n0 观察到 tag n0:1
    right.Add("deploy", "n1:9") // 分区期间并发 add，tag 未被观察

    _ = left.Merge(right.Clone())
    _ = right.Merge(left.Clone())
    fmt.Println(left.Elements()) // [deploy] —— add-wins
}
```

完整可运行示例见 [`examples/basic/main.go`](examples/basic/main.go)。

### 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 全部类型收敛、三条代数律与全部不变量通过 |
| `1` | 存在未收敛、不变量违反或运行错误 |
| `2` | 参数 / 用法错误 |

因此可直接写进脚本或 CI：

```bash
go run ./cmd/confluxlab verify -rounds 500 -replicas 5 || exit 1
```

### 常见问题

**Q：不同种子结果不同，是不是不稳定？**
A：恰恰相反。种子决定整条调度序列，同种子逐位一致；多种子用于扩大覆盖。CI 里同时跑多 seed 与传统单测。

**Q：`MVRegister` 的快照里出现多个值，是 bug 吗？**
A：不是。这是设计取舍——它**不擅自丢弃**并发写入，把冲突显式交给上层。若需要自动选胜者，请改用 `LWWRegister`。

**Q：新增自己的 CRDT 会被验证器覆盖吗？**
A：会，按 [`CONTRIBUTING.md`](CONTRIBUTING.md) 的清单登记到 `AllKinds()` 后，`laws` 与 `verify` 自动纳入。

---

## 💡 设计思路与迭代规划

### 三条代数律为什么是收敛性的根基

状态型 CRDT 的 `Merge` 本质上是偏序集合上的**最小上界（join）**。只要它满足：

| 定律 | 含义 | 对应现实问题 |
| --- | --- | --- |
| 交换律 `a⊔b = b⊔a` | 结果与合并顺序无关 | 消息乱序到达 |
| 结合律 `(a⊔b)⊔c = a⊔(b⊔c)` | 结果与合并拓扑无关 | 星型 /  gossip / 链式同步 |
| 幂等律 `a⊔a = a` | 重复合并无副作用 | 消息重传、至少一次投递 |

三者共同构成 **join-semilattice**，副本无需任何协调即可最终收敛到同一状态。ConfluxLab 没有把这三条写在注释里，而是写成了**每次 CI 都执行的随机化回归测试**。

### 沙盘的事件模型

```text
副本状态 × N + 向量时钟 × N + 分区分组 × N
        ↓ 每轮随机选择一种事件
  ├─ 本地写入（45%）  → 推进本地 Lamport 时戳与向量时钟，施加 Op
  ├─ 双向同步（43%）  → 分区内允许；跨分区记「受阻」；含概率重复投递
  └─ 分区切换（12%）  → 把某个副本移到另一分区
        ↓ 调度结束
  网络愈合：撤销全部分区，反复两两反熵合并直至稳定
        ↓
  逐副本快照比对 + 单调性 / 墓碑完整性检查 + 结构化报告
```

### 关键设计决策

1. **确定性优先于"更随机"**：所有随机性来自 `RNG`，`time.Now()` 不参与任何调度决策。失败时只需一个种子即可完整回放，这在调试收敛问题时价值极高。
2. **写入标识全局唯一是硬前提**：`LWWRegister` 的 `(时戳, 节点)`、`ORSet` 的 tag 必须唯一。开发过程中代数律验证器正是抓到了生成器违反该前提导致的交换律假失败；现在 `Merge` 会显式报错而不是静默挑选（详见 [`CHANGELOG.md`](CHANGELOG.md)）。
3. **冲突显式化**：`Merge` 不允许静默丢信息。要么全序确定胜者，要么保留全部并发版本。
4. **零依赖**：不引入 CLI 框架、测试框架或断言库，降低供应链风险，也让 `go test ./...` 在任何环境都能直接跑。

### 后续迭代方向

1. 序列型 CRDT（RGA / LSEQ）文本协同与墓碑压缩
2. 操作型 delta 传输与带宽统计，量化反熵全量交换的代价
3. 最小复现序列自动收缩（delta debugging）：失败时输出最短调度
4. 拜占庭副本注入：模拟恶意节点发送非法状态
5. 导出 JSON Schema，让编辑器可补全报告字段
6. 与真实网络层（如 WebSocket / NATS）对接做端到端验证

> 更完整的推导见 [`docs/design.md`](docs/design.md)。

---

## 📦 打包与部署指南

本项目是**工具库 + CLI**（不是需要安装器的桌面应用），提供两种交付形态。

### 作为库引入

```bash
go get github.com/gitstq/ConfluxLab
```

无外部依赖，无需额外配置；模块路径 `github.com/gitstq/ConfluxLab`。

### 全平台静态二进制

```bash
make cross                  # 等价于 bash scripts/build.sh
# 或指定版本
VERSION=v1.0.0 bash scripts/build.sh
```

产出目录 `dist/`：

| 目标 | 文件 |
| --- | --- |
| Linux amd64 / arm64 | `confluxlab-v1.0.0-linux-amd64` / `...-linux-arm64` |
| macOS amd64 / arm64 | `confluxlab-v1.0.0-darwin-amd64` / `...-darwin-arm64` |
| Windows amd64 | `confluxlab-v1.0.0-windows-amd64.exe` |
| 校验和 | `SHA256SUMS.txt` |

```bash
# 校验
sha256sum -c dist/SHA256SUMS.txt

# 直接运行（Linux 示例，静态链接，无运行时依赖）
./dist/confluxlab-v1.0.0-linux-amd64 verify -rounds 500
```

> ⚠️ macOS / Windows 产物由 Linux 交叉编译生成，构建与文件格式已校验，但运行时行为未在对应平台实机验证；请在目标平台跑一次 `version` 与 `verify` 冒烟。

### 接入 CI

仓库自带 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)：Go 1.21/1.22/1.23 三版本矩阵，依次执行格式检查、`go vet`、`go test -race`、构建、代数律门禁、5 副本 500 事件收敛性门禁、确定性回放 diff，以及**硬编码凭据扫描**。任意一步失败即阻断合并。

在非 GitHub 的流水线里，最小接入方式是两条命令：

```bash
go test ./... -count=1
go run ./cmd/confluxlab verify -rounds 500 -replicas 5
```

---

## 🤝 贡献指南

欢迎新增 CRDT 类型、加强不变量检查或补充语言版本。请先阅读 [`CONTRIBUTING.md`](CONTRIBUTING.md)，要点：

- 提交信息遵循 **Angular 规范**：`feat:` / `fix:` / `docs:` / `refactor:` / `test:`
- 新增类型须实现 `State` 接口、登记 `AllKinds()`、写单元测试，并跑通 `make all`
- 三条红线：**不引入运行时依赖**、**确定性优先**、**冲突显式化**
- 提交 Issue 请附 `-seed / -rounds / -replicas` 与 `-json` 输出，便于精确复现

---

## 📄 开源协议说明

本项目采用 [MIT License](LICENSE)。

- ✅ 允许商用、修改、分发、再授权
- ✅ 无需支付费用，无附加限制
- ⚠️ 需保留版权声明与许可声明
- ⚠️ 软件按"原样"提供，不含任何明示或暗示担保

完整条款见 [`LICENSE`](LICENSE) 文件。Copyright (c) 2026 gitstq。
