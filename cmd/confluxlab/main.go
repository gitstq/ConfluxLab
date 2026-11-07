// Command confluxlab 是 ConfluxLab 的命令行入口。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gitstq/ConfluxLab"
)

// version 可通过 -ldflags "-X main.version=..." 在发布时注入。
var version = "v1.0.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "verify":
		cmdVerify(os.Args[2:])
	case "laws":
		cmdLaws(os.Args[2:])
	case "demo":
		cmdDemo()
	case "explain":
		cmdExplain()
	case "version":
		fmt.Printf("ConfluxLab %s · 零依赖 CRDT 收敛性工具箱\n", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Print(`ConfluxLab · 把「收敛性」变成可测量、可回归测试的指标

用法:
  confluxlab verify   [选项]   运行随机并发沙盘，验证多副本最终收敛
  confluxlab laws     [选项]   验证交换律 / 结合律 / 幂等律
  confluxlab demo              演示 add-wins、LWW 平局打破与并发冲突暴露
  confluxlab explain           输出内置 CRDT 类型与代数律说明
  confluxlab version           输出版本号

verify 选项:
  -seed uint        随机种子（默认 20261108，同种子必得逐位一致结果）
  -rounds int       每种类型的调度事件数（默认 400）
  -replicas int     副本数（默认 4）
  -kinds string     逗号分隔的类型，all 表示全部
  -json             输出机器可读 JSON 报告
  -timeline         输出事件时间线

laws 选项:
  -seed uint        随机种子
  -rounds int       每种类型的随机状态组数（默认 200）
  -kinds string     逗号分隔的类型

退出码: 0=全部通过，1=存在不收敛或不变量违反，2=用法错误。
`)
}

func parseKinds(raw string) ([]string, error) {
	if raw == "" || raw == "all" {
		return confluxlab.AllKinds(), nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "all" {
			return confluxlab.AllKinds(), nil
		}
		if _, err := confluxlab.NewState(p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("未指定任何有效的 CRDT 类型")
	}
	return out, nil
}

func cmdVerify(args []string) {
	cfg := confluxlab.DefaultConfig()
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	seed := fs.Uint64("seed", cfg.Seed, "随机种子")
	rounds := fs.Int("rounds", cfg.Rounds, "调度事件数")
	replicas := fs.Int("replicas", cfg.Replicas, "副本数")
	kindsRaw := fs.String("kinds", "all", "逗号分隔的 CRDT 类型")
	jsonOut := fs.Bool("json", false, "输出 JSON 报告")
	timeline := fs.Bool("timeline", false, "输出事件时间线")
	_ = fs.Parse(args)

	kinds, err := parseKinds(*kindsRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "参数错误:", err)
		os.Exit(2)
	}
	cfg.Seed, cfg.Rounds, cfg.Replicas, cfg.Kinds = *seed, *rounds, *replicas, kinds
	report, err := confluxlab.RunSandbox(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行失败:", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "序列化失败:", err)
			os.Exit(1)
		}
	} else {
		printVerifyTable(report, *timeline)
	}
	if !report.Ok() {
		os.Exit(1)
	}
}

func printVerifyTable(r confluxlab.SandboxReport, showTimeline bool) {
	fmt.Println("ConfluxLab · 随机并发收敛性验证")
	fmt.Println(strings.Repeat("─", 78))
	fmt.Printf("  %s\n", r.Summary())
	fmt.Println(strings.Repeat("─", 78))
	fmt.Printf("  %-14s %-8s %8s %8s %8s %8s %8s %6s\n",
		"类型", "收敛", "本地操作", "合并", "重复", "受阻", "分区", "违反")
	for _, k := range r.Kinds {
		mark := "✅ 是"
		if !k.Ok() {
			mark = "❌ 否"
		}
		fmt.Printf("  %-14s %-8s %8d %8d %8d %8d %8d %6d\n",
			k.Kind, mark, k.LocalOps, k.Merges, k.Duplicates, k.BlockedSyncs,
			k.PartitionChanges, len(k.Violations))
	}
	fmt.Println(strings.Repeat("─", 78))
	for _, k := range r.Kinds {
		fmt.Printf("  %-14s 最终状态: %s\n", k.Kind, k.FinalSnapshot)
	}
	if len(r.Violations) > 0 {
		fmt.Println("  违反项:")
		for _, v := range r.Violations {
			fmt.Println("    -", v)
		}
	}
	if showTimeline {
		fmt.Println(strings.Repeat("─", 78))
		for _, k := range r.Kinds {
			fmt.Print(confluxlab.RenderTimeline(k))
		}
	}
	status := "全部收敛，三条代数律与全部不变量通过 ✅"
	if !r.Ok() {
		status = "存在未收敛或不变量违反 ❌"
	}
	fmt.Println(strings.Repeat("─", 78))
	fmt.Println("  结论:", status)
}

func cmdLaws(args []string) {
	fs := flag.NewFlagSet("laws", flag.ExitOnError)
	seed := fs.Uint64("seed", 20261108, "随机种子")
	rounds := fs.Int("rounds", 200, "随机状态组数")
	kindsRaw := fs.String("kinds", "all", "逗号分隔的 CRDT 类型")
	_ = fs.Parse(args)

	kinds, err := parseKinds(*kindsRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "参数错误:", err)
		os.Exit(2)
	}
	results, err := confluxlab.CheckLaws(*seed, *rounds, kinds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行失败:", err)
		os.Exit(1)
	}
	fmt.Println("ConfluxLab · join-semilattice 代数律验证")
	fmt.Println(strings.Repeat("─", 64))
	fmt.Printf("  %-14s %12s %10s %s\n", "类型", "检查次数", "违反", "结果")
	ok := true
	for _, r := range results {
		mark := "✅ 通过"
		if !r.Ok() {
			mark = "❌ 失败"
			ok = false
		}
		fmt.Printf("  %-14s %12d %10d %s\n", r.Kind, r.Checked, len(r.Violations), mark)
		for _, v := range r.Violations {
			fmt.Println("      -", v)
		}
	}
	fmt.Println(strings.Repeat("─", 64))
	fmt.Println("  交换律 a⊔b = b⊔a ｜ 结合律 (a⊔b)⊔c = a⊔(b⊔c) ｜ 幂等律 a⊔a = a")
	if !ok {
		os.Exit(1)
	}
}

func cmdDemo() {
	fmt.Println("ConfluxLab · 三个真实并发语义演示")
	fmt.Println(strings.Repeat("─", 72))

	// 1) ORSet 的 add-wins：分区两侧分别 add 与 remove。
	a, b := confluxlab.NewORSet(), confluxlab.NewORSet()
	a.Add("task", "n0:1")
	b.Merge(a.Clone())
	a.Remove("task")      // n0 观察到 tag n0:1 并删除
	b.Add("task", "n1:7") // n1 在分区期间并发加入同一元素（新 tag）
	a.Merge(b.Clone())
	b.Merge(a.Clone())
	fmt.Println("  ① ORSet add-wins：分区两侧分别对同一元素执行 remove 与 add")
	fmt.Printf("     合并后可见元素 = %v  → 并发 add 存活（新 tag 未被观察到）\n", a.Elements())

	// 2) LWWRegister 的确定性平局打破。
	l0, l1 := confluxlab.NewLWWRegister(), confluxlab.NewLWWRegister()
	l0.Set("来自 n0", 5, "n0")
	l1.Set("来自 n1", 5, "n1")
	x, y := l0.Clone(), l1.Clone()
	_ = x.Merge(l1.Clone())
	_ = y.Merge(l0.Clone())
	vx, _ := x.(*confluxlab.LWWRegister).Get()
	vy, _ := y.(*confluxlab.LWWRegister).Get()
	fmt.Println("  ② LWWRegister 平局打破：相同时戳的两次并发写入")
	fmt.Printf("     两个副本独立合并后均得到 %q（节点名字典序决定，结果确定）\n", vx)
	fmt.Printf("     一致性 = %t\n", vx == vy)

	// 3) MVRegister 显式暴露冲突。
	m0, m1 := confluxlab.NewMVRegister(), confluxlab.NewMVRegister()
	c0, c1 := confluxlab.NewVectorClock(), confluxlab.NewVectorClock()
	c0["n0"] = 1
	c1["n1"] = 1
	m0.SetWithClock("配置 A", c0)
	m1.SetWithClock("配置 B", c1)
	m0.Merge(m1.Clone())
	fmt.Println("  ③ MVRegister 不静默丢弃冲突：两个并发写入同时保留")
	fmt.Printf("     并发版本 = %v  → 由上层决策，而非悄悄覆盖\n", m0.Values())
	fmt.Println(strings.Repeat("─", 72))
}

func cmdExplain() {
	fmt.Print(`ConfluxLab 内置 5 种状态型 CRDT（全部零第三方依赖，纯 Go 标准库）

  gcounter     只增计数器        每个节点一个分量，合并取逐分量 max
  pncounter    可增减计数器      两个只增计数器之差，支持减量
  lwwregister  最后写入者获胜    (逻辑时戳, 节点 ID) 全序，平局可确定打破
  mvregister   多值寄存器        保留所有并发版本，把冲突显式交给上层
  orset        观察-删除集合     唯一 tag + tombstone，并发 add 与 remove 为 add-wins

配套组件
  VectorClock  向量时钟，判定 before / after / equal / concurrent 四种因果序
  RNG          SplitMix64 确定性发生器，同种子必得逐位一致的调度序列

为什么这三条定律重要
  交换律保证消息乱序无所谓；结合律保证任意合并拓扑都等价；
  幂等律保证消息重复投递无副作用。三者共同构成 join-semilattice，
  是「无需协调即可最终收敛」的充分条件。ConfluxLab 把它们做成可执行的回归测试。

沙盘做了什么
  随机交错本地写入、双向同步、网络分区与消息重复投递，
  最后自动愈合分区并逐副本比对快照，同时检查单调性与 tombstone 完整性。
`)
}
