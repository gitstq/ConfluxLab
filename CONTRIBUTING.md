# 🤝 贡献指南

感谢你愿意让 ConfluxLab 变得更好。本项目刻意保持**零第三方运行时依赖**，所有新增能力都必须能用 Go 标准库实现。

## 环境要求

- Go ≥ 1.21（CI 覆盖 1.21 / 1.22 / 1.23）
- 无需 `go mod download`，仓库没有外部依赖

```bash
git clone https://github.com/gitstq/ConfluxLab.git
cd ConfluxLab
make all   # 格式检查 → vet → 单测 → 构建 → 代数律门禁 → 收敛性门禁
```

## 提交规范

严格遵循 Angular 提交规范：

```
feat: 新增 xxx CRDT 类型
fix: 修复 xxx 在分区场景下的收敛问题
docs: 补充繁中 README 的部署章节
refactor: 抽出 pruneMaximal 的公共逻辑
test: 增加 6 副本随机调度回归用例
```

## 新增一种 CRDT 的检查清单

1. 实现 `State` 接口：`Kind / Clone / Merge / Snapshot / Apply`。
2. 在 `state.go` 的 `AllKinds()` 与 `NewState()` 中登记。
3. **写入标识必须全局唯一**：`LWWRegister` 的 `(时戳, 节点)`、`ORSet` 的 tag 都不能碰撞。`Merge` 遇到同标识不同值必须返回错误，不得静默挑选。
4. `Merge` 必须满足交换律、结合律、幂等律，`laws.go` 会自动验证。
5. 在 `sandbox.go` 的 `checkInvariants` 中登记该类型的特有单调量。
6. 补充单元测试（`*_test.go`）与 `demo` 中的语义演示。
7. 跑通 `make all`，并确认 `go run ./cmd/confluxlab verify -rounds 500 -replicas 5` 全绿。

## 三条设计红线

- **不引入运行时依赖**：任何 import 都必须来自 Go 标准库。
- **确定性优先**：所有随机性来自 `RNG`，禁止使用 `math/rand` 的全局源或 `time.Now()` 参与调度决策；同种子必须可逐位回放。
- **冲突显式化**：`Merge` 不得静默丢弃信息。要么按全序确定胜者（LWW），要么把并发版本都保留下来交给上层（MV），并在文档中写清取舍。

## 反馈与问题

提交 Issue 时请附上 `-seed`、`-rounds`、`-replicas` 与 `-json` 输出，任何失败都应能凭种子精确复现。
