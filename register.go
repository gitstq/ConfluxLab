package confluxlab

import (
	"fmt"
	"sort"
)

// LWWRegister 是"最后写入者获胜"寄存器。
// 以 (逻辑时戳, 节点 ID) 构成全序：时戳大者胜，时戳相同时以节点名字典序打破平局。
// 该设计保证任意两个副本对同一组写入必然选出同一个值。
type LWWRegister struct {
	value string
	stamp uint64
	node  string
	set   bool
}

// NewLWWRegister 创建空寄存器。
func NewLWWRegister() *LWWRegister { return &LWWRegister{} }

// Kind 返回类型名。
func (r *LWWRegister) Kind() string { return "lwwregister" }

// Clone 返回独立副本。
func (r *LWWRegister) Clone() State { c := *r; return &c }

// Set 写入一个新值。
func (r *LWWRegister) Set(value string, stamp uint64, node string) {
	r.value, r.stamp, r.node, r.set = value, stamp, node, true
}

// Get 返回当前值；从未写入时返回空串与 false。
func (r *LWWRegister) Get() (string, bool) { return r.value, r.set }

// wins 报告候选 (stamp, node) 是否严格优于当前版本。
func (r *LWWRegister) wins(stamp uint64, node string) bool {
	if !r.set {
		return true
	}
	if stamp != r.stamp {
		return stamp > r.stamp
	}
	return node > r.node
}

// Merge 保留全序中最大的版本。
// 若两个版本的 (时戳, 节点) 完全相同却带不同值，说明写入标识不唯一，
// 这违背了 LWW 的前提假设，必须显式报错而不是静默选一个。
func (r *LWWRegister) Merge(s State) error {
	o, ok := s.(*LWWRegister)
	if !ok {
		return typeError(r.Kind(), s)
	}
	if o.set && r.set && o.stamp == r.stamp && o.node == r.node && o.value != r.value {
		return fmt.Errorf("LWWRegister 写入标识冲突: (%d,%s) 同时对应 %q 与 %q", o.stamp, o.node, r.value, o.value)
	}
	if o.set && r.wins(o.stamp, o.node) {
		r.value, r.stamp, r.node, r.set = o.value, o.stamp, o.node, true
	}
	return nil
}

// Snapshot 返回可比较的结构化快照。
func (r *LWWRegister) Snapshot() any {
	return struct {
		Value string
		Stamp uint64
		Node  string
		Set   bool
	}{r.value, r.stamp, r.node, r.set}
}

// Apply 施加本地操作。
func (r *LWWRegister) Apply(op Op) error {
	if op.Type == "set" {
		r.Set(op.Arg, op.Stamp, op.Node)
		return nil
	}
	return errUnsupported(op, r.Kind())
}

// mvEntry 是 MVRegister 中的一个并发版本。
type mvEntry struct {
	Value string
	Clock VectorClock
}

// MVRegister 是多值寄存器：不擅自丢弃并发写入，
// 而是保留所有"未被其他版本因果支配"的值，把冲突显式交给上层决策。
// 与 LWWRegister 的"静默覆盖"形成明确的工程取舍对照。
type MVRegister struct {
	entries []mvEntry
}

// NewMVRegister 创建空的多值寄存器。
func NewMVRegister() *MVRegister { return &MVRegister{entries: []mvEntry{}} }

// Kind 返回类型名。
func (r *MVRegister) Kind() string { return "mvregister" }

// Clone 返回独立副本（深拷贝向量时钟）。
func (r *MVRegister) Clone() State {
	out := &MVRegister{entries: make([]mvEntry, 0, len(r.entries))}
	for _, e := range r.entries {
		out.entries = append(out.entries, mvEntry{Value: e.Value, Clock: e.Clock.Clone()})
	}
	return out
}

// Set 以给定向量时钟写入新版本，并剔除被新版本因果支配的旧版本。
func (r *MVRegister) Set(value string, clock VectorClock) {
	kept := make([]mvEntry, 0, len(r.entries)+1)
	for _, e := range r.entries {
		rel := e.Clock.Compare(clock)
		if rel == RelationBefore || rel == RelationEqual {
			continue // 被新版本支配或就是同一版本
		}
		kept = append(kept, e)
	}
	kept = append(kept, mvEntry{Value: value, Clock: clock.Clone()})
	r.entries = pruneMaximal(kept)
}

// Values 返回当前所有并发值（已排序去重）。
func (r *MVRegister) Values() []string {
	seen := make(map[string]bool)
	for _, e := range r.entries {
		seen[e.Value] = true
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// pruneMaximal 去除重复条目并仅保留因果序上的极大版本。
func pruneMaximal(entries []mvEntry) []mvEntry {
	unique := make([]mvEntry, 0, len(entries))
	seen := make(map[string]bool)
	for _, e := range entries {
		key := e.Clock.String() + "|" + e.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, e)
	}
	maximal := make([]mvEntry, 0, len(unique))
	for i, e := range unique {
		dominated := false
		for j, f := range unique {
			if i == j {
				continue
			}
			if e.Clock.Compare(f.Clock) == RelationBefore {
				dominated = true
				break
			}
		}
		if !dominated {
			maximal = append(maximal, e)
		}
	}
	// 排序保证快照稳定，与遍历顺序无关。
	sort.Slice(maximal, func(i, j int) bool {
		a, b := maximal[i], maximal[j]
		if a.Clock.String() != b.Clock.String() {
			return a.Clock.String() < b.Clock.String()
		}
		return a.Value < b.Value
	})
	return maximal
}

// Merge 合并两个版本集合，再保留极大版本。
func (r *MVRegister) Merge(s State) error {
	o, ok := s.(*MVRegister)
	if !ok {
		return typeError(r.Kind(), s)
	}
	merged := append([]mvEntry{}, r.entries...)
	for _, e := range o.entries {
		merged = append(merged, mvEntry{Value: e.Value, Clock: e.Clock.Clone()})
	}
	r.entries = pruneMaximal(merged)
	return nil
}

// Snapshot 返回排序后的并发值列表；调用方可用 len() 判断是否发生冲突。
func (r *MVRegister) Snapshot() any {
	return struct{ Values []string }{Values: r.Values()}
}

// Apply 施加本地操作：以 (节点, 时戳) 构造向量时钟的一个极小版本。
// 沙盘会为副本维护真实向量时钟并通过 SetDirect 注入。
func (r *MVRegister) Apply(op Op) error {
	if op.Type != "set" {
		return errUnsupported(op, r.Kind())
	}
	clock := NewVectorClock()
	clock[op.Node] = op.Stamp
	r.Set(op.Arg, clock)
	return nil
}

// SetWithClock 允许外部（沙盘）传入真实的因果时钟写入。
func (r *MVRegister) SetWithClock(value string, clock VectorClock) { r.Set(value, clock) }

// Describe 返回该类型的中文说明。
func (r *MVRegister) Describe() string {
	return fmt.Sprintf("MVRegister 当前保留 %d 个并发版本", len(r.entries))
}
