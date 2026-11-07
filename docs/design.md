# 🧠 设计说明：为什么 ConfluxLab 能保证收敛

本文补充 README 中被压缩掉的推导过程，供希望扩展或审查实现的人阅读。

## 1. 收敛性的数学基础

状态型 CRDT 的本地状态集合 `S` 配上一个偏序 `≤`，`Merge` 被定义为**最小上界（join, ⊔）**。若 `⊔` 满足：

```
交换律   a ⊔ b = b ⊔ a
结合律   (a ⊔ b) ⊔ c = a ⊔ (b ⊔ c)
幂等律   a ⊔ a = a
```

则 `(S, ⊔)` 构成一个 **join-semilattice**。此时：

- 交换律 ⇒ 消息到达顺序无关；
- 结合律 ⇒ 合并拓扑（星型 / gossip / 链式）无关；
- 幂等律 ⇒ 重复投递无副作用。

只要所有副本最终都收到同一组更新（哪怕顺序、路径、次数各不相同），它们必然收敛到同一个最小上界。这正是"无需协调即可最终一致"的全部依据。

**ConfluxLab 的做法**：不把这三条当作注释里的假设，而是 `CheckLaws` 每次运行随机生成状态三元组 `(a, b, c)`，逐一验证三条等式。任何一条失败，`laws` 与 CI 直接失败。

## 2. 五种 CRDT 的状态与合并规则

| 类型 | 状态 | `⊔` 的定义 | 特有语义 |
| --- | --- | --- | --- |
| `GCounter` | `node → uint64` | 逐分量取 `max` | 只增，天然幂等 |
| `PNCounter` | `(P, N)` 两个 `GCounter` | 分别取 `max` | `Value = ΣP − ΣN`，可为负 |
| `LWWRegister` | `(value, stamp, node)` | 按 `(stamp, node)` 全序取最大 | 时戳相同用节点名字典序打破平局 |
| `MVRegister` | `[(value, VectorClock)]` | 并集后保留因果极大版本 | 并发写入全部保留，冲突交由上层 |
| `ORSet` | `adds: tag → elem`、`removed: set[tag]` | 两个集合取并集 | add-wins：`Remove` 只删除已观察到的 tag |

`MVRegister` 的 `pruneMaximal` 是关键：合并后遍历所有版本，剔除任何被另一版本严格支配的条目（即 `e.Clock.Compare(f.Clock) == RelationBefore`），再按 `(时钟字符串, 值)` 稳定排序，保证快照与切片遍历顺序无关。

## 3. 写入标识全局唯一：一个容易忽略的硬前提

全序与 tag 机制都隐含一个前提：**每次写入的标识全局唯一**。

- `LWWRegister` 依赖 `(stamp, node)` 唯一；若两个副本用同一 `(stamp, node)` 写了不同值，`a⊔b` 与 `b⊔a` 会各自保留"先到者"，交换律被破坏。
- `ORSet` 依赖 tag 唯一；若同一 tag 对应不同元素，合并方向会决定元素，交换律同样被破坏。

开发首轮 `CheckLaws` 就抓到了这个现象：`lwwregister` 与 `orset` 在若干随机轮次报交换律失败。排查后确认**不是实现缺陷，而是生成器让 a/b/c 各自从 1 开始编号，制造了非法输入**。修复分两步：

1. `randomState` 改用 a/b/c 共享、且跨轮次持续递增的全局时戳源 `stampCounter`；
2. 两个 `Merge` 增加冲突检测：同标识不同值时返回明确错误，而不是静默挑选一个。

这一步很有代表性：**验证器的价值不只在于发现实现 bug，也在于暴露被忽略的前提假设**。真实系统里，标识唯一性由「Lamport 时戳 + 节点 ID」或「UUID」保证；沙盘中由 `(副本 ID, 本地递增 Lamport 时戳)` 保证。

## 4. 向量时钟与因果序

`VectorClock.Compare` 遍历两方键的**并集**（缺失分量视为 0），统计"某分量更小"与"某分量更大"是否同时出现：

| 更小 | 更大 | 关系 |
| --- | --- | --- |
| ✓ | ✗ | `Before` |
| ✗ | ✓ | `After` |
| ✗ | ✗ | `Equal` |
| ✓ | ✓ | `Concurrent` |

早期实现只遍历左值键，导致 `{n0:1}` 与 `{n0:1, n1:1}` 被误判为 `Equal`；修复为并集遍历后已加入回归用例。

## 5. 沙盘事件模型与不变量

每轮按固定权重随机选择事件：

| 权重 | 事件 | 动作 |
| --- | --- | --- |
| 45% | 本地写入 | 推进本地 Lamport 时戳与向量时钟，施加 `Op` |
| 43% | 双向同步 | 同分区允许；跨分区记 `BlockedSyncs`；按概率重复投递 |
| 12% | 分区切换 | 翻转某副本的分区分组 |

同步使用**合并前的快照**做双向 `Merge`，并同步双方向量时钟：

```
si, sj := states[i].Clone(), states[j].Clone()
states[i].Merge(sj); states[j].Merge(si)
clocks[i].Merge(clocks[j].Clone()); clocks[j].Merge(clocks[i].Clone())
```

调度结束后撤销全部分区，反复两两反熵合并直到所有快照一致（最多 `2N+2` 轮）。

**不变量清单**

| 类型 | 检查项 |
| --- | --- |
| `GCounter` | `Value()` 单调不减 |
| `PNCounter` | `ΣP + ΣN` 单调不减（两个分量各自只增） |
| `ORSet` | `TagCount()` 单调不减；每个 tombstone 都有对应 add 记录 |
| 全部 | 愈合后所有副本 `Snapshot()` 深度相等 |

`LWWRegister` 与 `MVRegister` 无单调量，其正确性由全序确定性 / 极大版本保留 + 收敛比对覆盖。

## 6. 确定性设计

- 所有随机性来自 `RNG`（SplitMix64），不使用 `math/rand` 全局源，`time.Now()` 不参与任何调度决策。
- 所有可能受 map 遍历序影响的输出都做了稳定化：`ORSet.Elements()` 排序、`MVRegister.Values()` 排序、`pruneMaximal` 稳定排序、`VectorClock.String()` 按节点名排序。
- 因此 `RunSandbox(cfg)` 对同一 `cfg` 逐位一致，`-json` 输出可直接 `diff`。CI 中专门有一条确定性回放校验。

## 7. 已知边界

- 只覆盖**状态型** CRDT，未实现序列型（RGA / LSEQ）文本协同。
- 沙盘模拟全量状态交换（反熵），不含 delta 传输与带宽建模。
- `MVRegister` 保留冲突但不解决冲突。
- 分区模型是二分组的连通性阻断，不模拟消息延迟、乱序队列或拜占庭节点。
- macOS / Windows 二进制为交叉编译产物，未在对应平台实机运行验证。

这些边界都是**明确的、写进文档的取舍**，而不是隐藏限制。
