package confluxlab

import (
	"fmt"
	"strings"
)

// SandboxConfig 描述一次随机并发调度的参数。
type SandboxConfig struct {
	Seed            uint64
	Rounds          int
	Replicas        int
	Kinds           []string
	DuplicateChance float64 // 消息重复投递概率，用于压力测试幂等律
	TimelineLimit   int
}

// DefaultConfig 返回推荐的默认配置。
func DefaultConfig() SandboxConfig {
	return SandboxConfig{
		Seed:            20261108,
		Rounds:          400,
		Replicas:        4,
		Kinds:           AllKinds(),
		DuplicateChance: 0.18,
		TimelineLimit:   14,
	}
}

// KindReport 是单个 CRDT 类型的收敛性验证报告。
type KindReport struct {
	Kind             string
	Converged        bool
	Replicas         int
	LocalOps         int
	Merges           int
	Duplicates       int
	BlockedSyncs     int
	PartitionChanges int
	Violations       []string
	FinalSnapshot    string
	Timeline         []string
}

// Ok 报告该类型是否通过全部检查。
func (r KindReport) Ok() bool { return r.Converged && len(r.Violations) == 0 }

// SandboxReport 是一次完整沙盘运行的结构化结果，可直接序列化为 JSON。
type SandboxReport struct {
	Seed       uint64
	Rounds     int
	Replicas   int
	Converged  bool
	Violations []string `json:",omitempty"`
	Kinds      []KindReport
}

// Ok 报告整次运行是否全部收敛且无不变量违反。
func (r SandboxReport) Ok() bool { return r.Converged && len(r.Violations) == 0 }

// Summary 返回单行人类可读摘要。
func (r SandboxReport) Summary() string {
	ops, merges := 0, 0
	for _, k := range r.Kinds {
		ops += k.LocalOps
		merges += k.Merges
	}
	status := "CONVERGED"
	if !r.Ok() {
		status = "FAILED"
	}
	return fmt.Sprintf("seed=%d rounds=%d replicas=%d localOps=%d merges=%d kinds=%d status=%s",
		r.Seed, r.Rounds, r.Replicas, ops, merges, len(r.Kinds), status)
}

// RunSandbox 执行确定性随机并发调度：本地操作、双向合并、网络分区、
// 消息重复投递交错进行；随后自动愈合分区并验证所有副本收敛。
func RunSandbox(cfg SandboxConfig) (SandboxReport, error) {
	report := SandboxReport{Seed: cfg.Seed, Violations: []string{}, Kinds: []KindReport{}}
	if cfg.Replicas < 2 {
		cfg.Replicas = 3
	}
	if cfg.Rounds <= 0 {
		cfg.Rounds = 400
	}
	if len(cfg.Kinds) == 0 {
		cfg.Kinds = AllKinds()
	}
	if cfg.TimelineLimit <= 0 {
		cfg.TimelineLimit = 14
	}
	report.Rounds, report.Replicas = cfg.Rounds, cfg.Replicas

	n := cfg.Replicas
	nodes := make([]string, n)
	for i := range nodes {
		nodes[i] = fmt.Sprintf("n%d", i)
	}
	universe := []string{"alpha", "beta", "gamma", "delta", "epsilon"}

	for idx, kind := range cfg.Kinds {
		if _, err := NewState(kind); err != nil {
			return report, err
		}
		rng := NewRNG(cfg.Seed + uint64(idx+1)*0xBF58476D1CE4E5B9)
		states := make([]State, n)
		clocks := make([]VectorClock, n)
		lamport := make([]uint64, n)
		groups := make([]int, n) // 0/1 表示副本所属网络分区
		mono := make([]uint64, n)
		for i := range states {
			states[i], _ = NewState(kind)
			clocks[i] = NewVectorClock()
		}
		kr := KindReport{Kind: kind, Replicas: n, Violations: []string{}, Timeline: []string{}}
		addLine := func(s string) {
			if len(kr.Timeline) < cfg.TimelineLimit {
				kr.Timeline = append(kr.Timeline, s)
			}
		}

		for step := 0; step < cfg.Rounds; step++ {
			roll := rng.Next() % 100
			switch {
			case roll < 45: // 本地操作
				i := rng.Intn(n)
				lamport[i]++
				clocks[i].Tick(nodes[i])
				op := randomOp(kind, rng, nodes[i], lamport[i], universe)
				if kind == "mvregister" {
					states[i].(*MVRegister).SetWithClock(op.Arg, clocks[i])
				} else if err := states[i].Apply(op); err != nil {
					return report, err
				}
				kr.LocalOps++
				addLine(fmt.Sprintf("#%03d  本地写入  %-28s → %s", step, op.String(), nodes[i]))

			case roll < 88: // 双向同步（可能被分区阻断）
				i, j := rng.Intn(n), rng.Intn(n)
				if i == j {
					j = (j + 1) % n
				}
				if groups[i] != groups[j] {
					kr.BlockedSyncs++
					addLine(fmt.Sprintf("#%03d  同步受阻  %s ↔ %s  (分区 %d|%d)", step, nodes[i], nodes[j], groups[i], groups[j]))
					break
				}
				if err := mergePair(states, clocks, i, j); err != nil {
					return report, err
				}
				kr.Merges++
				if rng.Chance(cfg.DuplicateChance) {
					// 同一消息重复投递：幂等律保证不应改变任何状态。
					if err := mergePair(states, clocks, i, j); err != nil {
						return report, err
					}
					kr.Duplicates++
					addLine(fmt.Sprintf("#%03d  重复投递 %s ↔ %s  (验证幂等)", step, nodes[i], nodes[j]))
				} else {
					addLine(fmt.Sprintf("#%03d  双向同步  %s ↔ %s", step, nodes[i], nodes[j]))
				}

			default: // 分区切换
				i := rng.Intn(n)
				groups[i] = 1 - groups[i]
				kr.PartitionChanges++
				addLine(fmt.Sprintf("#%03d  分区切换  %s → 分区 %d", step, nodes[i], groups[i]))
			}

			for k := 0; k < n; k++ {
				checkInvariants(kind, states[k], mono, k, &kr.Violations)
			}
		}

		// 愈合：撤销所有分区，反复两两合并直至稳定。
		for k := range groups {
			groups[k] = 0
		}
		addLine("──── 网络愈合：全连通反熵 ────")
		for pass := 0; pass < n*2+2; pass++ {
			for i := 0; i < n; i++ {
				for j := i + 1; j < n; j++ {
					if err := mergePair(states, clocks, i, j); err != nil {
						return report, err
					}
				}
			}
			stable := true
			for i := 1; i < n; i++ {
				if !snapshotsEqual(states[0].Snapshot(), states[i].Snapshot()) {
					stable = false
					break
				}
			}
			if stable {
				break
			}
		}

		base := states[0].Snapshot()
		for i := 1; i < n; i++ {
			if !snapshotsEqual(base, states[i].Snapshot()) {
				kr.Violations = append(kr.Violations,
					fmt.Sprintf("副本 n0 与 %s 未收敛: %v != %v", nodes[i], base, states[i].Snapshot()))
			}
		}
		for k := 0; k < n; k++ {
			checkInvariants(kind, states[k], mono, k, &kr.Violations)
		}
		kr.Converged = len(kr.Violations) == 0
		kr.FinalSnapshot = snapshotString(base)
		addLine(fmt.Sprintf("收敛=%t  最终状态=%s", kr.Converged, kr.FinalSnapshot))
		report.Kinds = append(report.Kinds, kr)
		if !kr.Ok() {
			report.Violations = append(report.Violations, kr.Violations...)
		}
	}

	report.Converged = true
	for _, k := range report.Kinds {
		if !k.Ok() {
			report.Converged = false
			break
		}
	}
	return report, nil
}

// mergePair 基于合并前的快照做双向合并，并同步双方的向量时钟。
func mergePair(states []State, clocks []VectorClock, i, j int) error {
	si, sj := states[i].Clone(), states[j].Clone()
	ci, cj := clocks[i].Clone(), clocks[j].Clone()
	if err := states[i].Merge(sj); err != nil {
		return err
	}
	if err := states[j].Merge(si); err != nil {
		return err
	}
	clocks[i].Merge(cj)
	clocks[j].Merge(ci)
	return nil
}

// checkInvariants 检查类型特有的不变量（单调性与 tombstone 完整性）。
func checkInvariants(kind string, st State, mono []uint64, idx int, violations *[]string) {
	var cur uint64
	switch kind {
	case "gcounter":
		cur = st.(*GCounter).Value()
	case "pncounter":
		c := st.(*PNCounter)
		cur = c.p.Value() + c.n.Value()
	case "orset":
		s := st.(*ORSet)
		cur = uint64(s.TagCount())
		for _, v := range s.ValidateInvariants() {
			*violations = append(*violations, fmt.Sprintf("副本 n%d: %s", idx, v))
		}
	default:
		return
	}
	if cur < mono[idx] {
		*violations = append(*violations,
			fmt.Sprintf("副本 n%d 单调性被破坏: %d → %d", idx, mono[idx], cur))
	}
	mono[idx] = cur
}

// RenderTimeline 把某类型的事件时间线渲染成带边框的文本块。
func RenderTimeline(kr KindReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  [%s]  收敛=%t  本地操作=%d  合并=%d  重复=%d  受阻=%d  分区切换=%d\n",
		kr.Kind, kr.Converged, kr.LocalOps, kr.Merges, kr.Duplicates, kr.BlockedSyncs, kr.PartitionChanges))
	for _, line := range kr.Timeline {
		b.WriteString("  │ " + line + "\n")
	}
	return b.String()
}
