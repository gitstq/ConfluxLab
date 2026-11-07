package confluxlab

import (
	"fmt"
	"sort"
)

// ORSet 是"观察-删除"集合（observed-remove set）。
// 每次 Add 生成一个全局唯一 tag；Remove 只删除调用方当时观察到的 tag。
// 因此并发的 Add 与 Remove 相遇时，新 tag 未被观察到，元素得以保留（add-wins）。
type ORSet struct {
	adds    map[string]string // tag -> element
	removed map[string]bool   // 已被删除的 tag
}

// NewORSet 创建空集合。
func NewORSet() *ORSet {
	return &ORSet{adds: make(map[string]string), removed: make(map[string]bool)}
}

// Kind 返回类型名。
func (s *ORSet) Kind() string { return "orset" }

// Clone 返回独立副本。
func (s *ORSet) Clone() State {
	out := NewORSet()
	for t, e := range s.adds {
		out.adds[t] = e
	}
	for t := range s.removed {
		out.removed[t] = true
	}
	return out
}

// Add 以唯一 tag 加入元素。
func (s *ORSet) Add(element, tag string) { s.adds[tag] = element }

// Remove 删除当前可见的所有同名元素 tag。
func (s *ORSet) Remove(element string) {
	for tag, e := range s.adds {
		if e == element && !s.removed[tag] {
			s.removed[tag] = true
		}
	}
}

// Elements 返回当前可见元素（排序去重）。
func (s *ORSet) Elements() []string {
	seen := make(map[string]bool)
	for tag, e := range s.adds {
		if !s.removed[tag] {
			seen[e] = true
		}
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// Contains 报告元素当前是否可见。
func (s *ORSet) Contains(element string) bool {
	for _, e := range s.Elements() {
		if e == element {
			return true
		}
	}
	return false
}

// TagCount 返回累计加入过的 tag 数量，用于验证"已删除 tag 不得复活"。
func (s *ORSet) TagCount() int { return len(s.adds) }

// Merge 取 adds 与 removed 的并集；重复合并无副作用（幂等律）。
func (s *ORSet) Merge(other State) error {
	o, ok := other.(*ORSet)
	if !ok {
		return typeError(s.Kind(), other)
	}
	for tag, e := range o.adds {
		if existing, exists := s.adds[tag]; exists {
			// tag 必须全局唯一；同 tag 对应不同元素意味着写入标识冲突。
			if existing != e {
				return fmt.Errorf("ORSet tag 冲突: %s 同时对应 %q 与 %q（tag 必须全局唯一）", tag, existing, e)
			}
			continue
		}
		s.adds[tag] = e
	}
	for tag := range o.removed {
		s.removed[tag] = true
	}
	return nil
}

// Snapshot 返回可见元素列表与累计 tag 数，便于跨副本比较。
func (s *ORSet) Snapshot() any {
	return struct {
		Elements []string
		Tags     int
	}{Elements: s.Elements(), Tags: len(s.adds)}
}

// Apply 施加本地操作；tag 由 (节点, 时戳) 合成，保证跨副本唯一。
func (s *ORSet) Apply(op Op) error {
	switch op.Type {
	case "add":
		s.Add(op.Arg, fmt.Sprintf("%s:%d", op.Node, op.Stamp))
		return nil
	case "remove":
		s.Remove(op.Arg)
		return nil
	default:
		return errUnsupported(op, s.Kind())
	}
}

// ValidateInvariants 检查 ORSet 内部一致性：任何已删除 tag 都不得重新变为可见。
func (s *ORSet) ValidateInvariants() []string {
	var violations []string
	for tag := range s.removed {
		if _, exists := s.adds[tag]; !exists {
			violations = append(violations, fmt.Sprintf("已删除 tag %s 缺少对应的 add 记录", tag))
		}
	}
	return violations
}
