package confluxlab

import "sort"

// GCounter 是只增计数器（grow-only counter）：每个节点持有独立分量，
// 合并时逐分量取最大值，因此天然幂等、可交换、可结合。
type GCounter struct {
	parts map[string]uint64
}

// NewGCounter 创建空的只增计数器。
func NewGCounter() *GCounter { return &GCounter{parts: make(map[string]uint64)} }

// Kind 返回类型名。
func (g *GCounter) Kind() string { return "gcounter" }

// Clone 返回独立副本。
func (g *GCounter) Clone() State {
	m := make(map[string]uint64, len(g.parts))
	for k, v := range g.parts {
		m[k] = v
	}
	return &GCounter{parts: m}
}

// Merge 逐分量取最大值。
func (g *GCounter) Merge(s State) error {
	o, ok := s.(*GCounter)
	if !ok {
		return typeError(g.Kind(), s)
	}
	for k, v := range o.parts {
		if v > g.parts[k] {
			g.parts[k] = v
		}
	}
	return nil
}

// Snapshot 返回分量的独立副本，便于跨副本比较。
func (g *GCounter) Snapshot() any {
	m := make(map[string]uint64, len(g.parts))
	for k, v := range g.parts {
		m[k] = v
	}
	return m
}

// Inc 在指定节点上累加 delta。
func (g *GCounter) Inc(node string, delta uint64) { g.parts[node] = g.parts[node] + delta }

// Value 返回所有分量之和。
func (g *GCounter) Value() uint64 {
	var total uint64
	for _, v := range g.parts {
		total += v
	}
	return total
}

// Apply 施加本地操作。
func (g *GCounter) Apply(op Op) error {
	switch op.Type {
	case "inc":
		g.Inc(op.Node, 1)
		return nil
	default:
		return errUnsupported(op, g.Kind())
	}
}

// PNCounter 是可增可减计数器，由两个 GCounter（增量 P 与减量 N）组成。
type PNCounter struct {
	p *GCounter
	n *GCounter
}

// NewPNCounter 创建空的可增减计数器。
func NewPNCounter() *PNCounter {
	return &PNCounter{p: NewGCounter(), n: NewGCounter()}
}

// Kind 返回类型名。
func (c *PNCounter) Kind() string { return "pncounter" }

// Clone 返回独立副本。
func (c *PNCounter) Clone() State {
	return &PNCounter{p: c.p.Clone().(*GCounter), n: c.n.Clone().(*GCounter)}
}

// Merge 分别合并两个分量计数器。
func (c *PNCounter) Merge(s State) error {
	o, ok := s.(*PNCounter)
	if !ok {
		return typeError(c.Kind(), s)
	}
	if err := c.p.Merge(o.p); err != nil {
		return err
	}
	return c.n.Merge(o.n)
}

// Snapshot 返回可比较的结构化快照。
func (c *PNCounter) Snapshot() any {
	return struct {
		P map[string]uint64
		N map[string]uint64
	}{P: c.p.Snapshot().(map[string]uint64), N: c.n.Snapshot().(map[string]uint64)}
}

// Inc 增加计数。
func (c *PNCounter) Inc(node string, delta uint64) { c.p.Inc(node, delta) }

// Dec 减少计数。
func (c *PNCounter) Dec(node string, delta uint64) { c.n.Inc(node, delta) }

// Value 返回 P 与 N 之差（以 int64 表示，允许为负）。
func (c *PNCounter) Value() int64 {
	return int64(c.p.Value()) - int64(c.n.Value())
}

// Apply 施加本地操作。
func (c *PNCounter) Apply(op Op) error {
	switch op.Type {
	case "inc":
		c.Inc(op.Node, 1)
		return nil
	case "dec":
		c.Dec(op.Node, 1)
		return nil
	default:
		return errUnsupported(op, c.Kind())
	}
}

// sortedKeys 返回按字典序排序的映射键，用于稳定输出。
func sortedKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
