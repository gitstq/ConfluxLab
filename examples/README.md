# 📁 示例目录

## `basic/main.go`

最小可运行示例，演示两件事：

1. **ORSet 的 add-wins 语义**：两个副本在分区期间分别对同一元素执行 `remove` 与 `add`，合并后元素依然可见。
2. **把沙盘当作回归测试调用**：`RunSandbox` 返回结构化报告与单行摘要。

```bash
cd ConfluxLab
go run ./examples/basic
```

输出示例：

```text
合并后可见元素: [deploy]
两副本是否一致: true
seed=20261108 rounds=200 replicas=4 localOps=... merges=... kinds=5 status=CONVERGED
```

## 更多用法

- CLI 全部子命令与参数见 [README](../README.md#-详细使用指南)。
- 五种 CRDT 作为库的 API 可直接阅读包内源码：`counter.go`、`register.go`、`orset.go`、`clock.go`、`laws.go`、`sandbox.go`。
- 新增自定义 CRDT 的清单见 [CONTRIBUTING](../CONTRIBUTING.md)。
