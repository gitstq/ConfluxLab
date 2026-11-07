package confluxlab

import "testing"

func TestGCounter(t *testing.T) {
	a, b := NewGCounter(), NewGCounter()
	a.Inc("n0", 3)
	a.Inc("n1", 2)
	b.Inc("n0", 1)
	b.Inc("n2", 8)
	if a.Value() != 5 || b.Value() != 9 {
		t.Fatal("初始值计算错误")
	}
	_ = a.Merge(b.Clone())
	_ = b.Merge(a.Clone()) // 注意：a 已在上一行被修改，这里仅验证不 panic
	if a.Value() != 13 {
		t.Fatalf("合并后应为 3+2+8=13，实际 %d", a.Value())
	}
	before := a.Snapshot()
	_ = a.Merge(a.Clone())
	if !snapshotsEqual(before, a.Snapshot()) {
		t.Fatal("重复合并幂等律失败")
	}
	_ = a.Apply(Op{Type: "inc", Node: "n0", Stamp: 1})
	if a.Value() != 14 {
		t.Fatal("Apply inc 未生效")
	}
}

func TestPNCounter(t *testing.T) {
	a := NewPNCounter()
	a.Inc("n0", 5)
	a.Dec("n0", 2)
	if a.Value() != 3 {
		t.Fatalf("Value 应为 3，实际 %d", a.Value())
	}
	b := NewPNCounter()
	b.Dec("n1", 4)
	_ = a.Merge(b.Clone())
	if a.Value() != -1 {
		t.Fatalf("合并后应为 5-2-4=-1，实际 %d", a.Value())
	}
	before := a.Snapshot()
	_ = a.Merge(b.Clone())
	if !snapshotsEqual(before, a.Snapshot()) {
		t.Fatal("重复合并应无副作用")
	}
}

func TestLWWRegisterTieBreak(t *testing.T) {
	a, b := NewLWWRegister(), NewLWWRegister()
	a.Set("value-A", 5, "n0")
	b.Set("value-B", 5, "n1")
	x, y := a.Clone(), b.Clone()
	_ = x.Merge(b.Clone())
	_ = y.Merge(a.Clone())
	vx, _ := x.(*LWWRegister).Get()
	vy, _ := y.(*LWWRegister).Get()
	if vx != vy || vx != "value-B" {
		t.Fatalf("平局必须由节点名字典序确定打破，实际 %q / %q", vx, vy)
	}
	// 更大时戳覆盖，与节点名无关。
	c := NewLWWRegister()
	c.Set("value-C", 6, "n0")
	_ = x.Merge(c.Clone())
	vx, _ = x.(*LWWRegister).Get()
	if vx != "value-C" {
		t.Fatal("更大时戳应获胜")
	}
	// 空寄存器不覆盖已有值。
	_ = x.Merge(NewLWWRegister().Clone())
	vx, _ = x.(*LWWRegister).Get()
	if vx != "value-C" {
		t.Fatal("空寄存器不应覆盖已有值")
	}
}

func TestMVRegisterConcurrentValues(t *testing.T) {
	a, b := NewMVRegister(), NewMVRegister()
	ca, cb := NewVectorClock(), NewVectorClock()
	ca["n0"] = 1
	cb["n1"] = 1
	a.SetWithClock("配置 A", ca)
	b.SetWithClock("配置 B", cb)
	_ = a.Merge(b.Clone())
	values := a.Values()
	if len(values) != 2 || values[0] != "配置 A" || values[1] != "配置 B" {
		t.Fatalf("并发写入应同时保留，实际 %v", values)
	}
	// 因果上更晚的写入应支配并清除旧版本。
	cc := NewVectorClock()
	cc["n0"] = 2
	cc["n1"] = 1
	a.SetWithClock("配置 C", cc)
	values = a.Values()
	if len(values) != 1 || values[0] != "配置 C" {
		t.Fatalf("因果更晚的写入应清除被支配版本，实际 %v", values)
	}
	// 合并顺序无关。
	p, q := NewMVRegister(), NewMVRegister()
	p.SetWithClock("A", ca)
	q.SetWithClock("B", cb)
	p1, q1 := p.Clone(), q.Clone()
	_ = p1.Merge(q.Clone())
	_ = q1.Merge(p.Clone())
	if !snapshotsEqual(p1.Snapshot(), q1.Snapshot()) {
		t.Fatal("MVRegister 交换律失败")
	}
}

func TestORSetAddWins(t *testing.T) {
	a, b := NewORSet(), NewORSet()
	a.Add("task", "n0:1")
	_ = b.Merge(a.Clone())
	a.Remove("task")
	b.Add("task", "n1:7") // 分区期间的并发 add，tag 未被 n0 观察到
	_ = a.Merge(b.Clone())
	_ = b.Merge(a.Clone())
	if !a.Contains("task") || !b.Contains("task") {
		t.Fatalf("add-wins 语义失败，实际 %v / %v", a.Elements(), b.Elements())
	}
	if len(a.ValidateInvariants()) != 0 {
		t.Fatal("ORSet 内部不变量被破坏")
	}
	// 被观察后再删除，元素应消失。
	a.Remove("task")
	if a.Contains("task") {
		t.Fatal("观察到所有 tag 后删除应生效")
	}
	before := a.Snapshot()
	_ = a.Merge(b.Clone())
	_ = a.Merge(b.Clone())
	if !snapshotsEqual(before, a.Snapshot()) {
		t.Fatal("重复合并幂等律失败")
	}
}

func TestTypeMismatchAndUnknown(t *testing.T) {
	g := NewGCounter()
	if err := g.Merge(NewORSet()); err == nil {
		t.Fatal("跨类型合并必须报错")
	}
	if _, err := NewState("nonexistent"); err == nil {
		t.Fatal("未知类型必须报错")
	}
	if err := g.Apply(Op{Type: "remove", Node: "n0"}); err == nil {
		t.Fatal("不支持的操作必须报错")
	}
}
