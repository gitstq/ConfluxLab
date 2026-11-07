package confluxlab

import (
	"fmt"
	"sort"
	"strings"
)

// Relation 描述两个向量时钟之间的因果序关系。
type Relation int

const (
	// RelationBefore 表示当前时钟因果上早于对方。
	RelationBefore Relation = iota
	// RelationAfter 表示当前时钟因果上晚于对方。
	RelationAfter
	// RelationEqual 表示两个时钟逐分量相等。
	RelationEqual
	// RelationConcurrent 表示两者并发，存在真实冲突。
	RelationConcurrent
)

// String 返回关系的可读名称。
func (r Relation) String() string {
	switch r {
	case RelationBefore:
		return "before"
	case RelationAfter:
		return "after"
	case RelationEqual:
		return "equal"
	default:
		return "concurrent"
	}
}

// VectorClock 是节点 ID 到逻辑计数的映射，用于判定事件的因果序。
type VectorClock map[string]uint64

// NewVectorClock 创建空时钟。
func NewVectorClock() VectorClock { return make(VectorClock) }

// Clone 返回独立副本。
func (vc VectorClock) Clone() VectorClock {
	out := make(VectorClock, len(vc))
	for k, v := range vc {
		out[k] = v
	}
	return out
}

// Tick 推进指定节点的本地计数。
func (vc VectorClock) Tick(node string) { vc[node] = vc[node] + 1 }

// Merge 把另一时钟逐分量取最大值（并集语义）。
func (vc VectorClock) Merge(other VectorClock) {
	for k, v := range other {
		if v > vc[k] {
			vc[k] = v
		}
	}
}

// Compare 判定当前时钟与 other 的因果关系。
// 同时出现"某分量更小"与"某分量更大"即为并发。
func (vc VectorClock) Compare(other VectorClock) Relation {
	keys := make(map[string]bool, len(vc)+len(other))
	for k := range vc {
		keys[k] = true
	}
	for k := range other {
		keys[k] = true
	}
	less, greater := false, false
	for k := range keys {
		a, b := vc[k], other[k]
		if a < b {
			less = true
		} else if a > b {
			greater = true
		}
	}
	switch {
	case less && !greater:
		return RelationBefore
	case greater && !less:
		return RelationAfter
	case less && greater:
		return RelationConcurrent
	default:
		return RelationEqual
	}
}

// HappensBefore 报告当前事件是否因果上早于 other。
func (vc VectorClock) HappensBefore(other VectorClock) bool {
	return vc.Compare(other) == RelationBefore
}

// Concurrent 报告两个事件是否并发。
func (vc VectorClock) Concurrent(other VectorClock) bool {
	return vc.Compare(other) == RelationConcurrent
}

// Dominates 报告当前时钟是否严格支配 other（即 other 因果上更早）。
func (vc VectorClock) Dominates(other VectorClock) bool {
	return other.Compare(vc) == RelationBefore
}

// String 输出稳定的、可比较的字符串表示（分量按节点名排序）。
func (vc VectorClock) String() string {
	keys := make([]string, 0, len(vc))
	for k := range vc {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", k, vc[k]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
