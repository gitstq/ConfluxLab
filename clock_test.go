package confluxlab

import "testing"

func TestVectorClockCompare(t *testing.T) {
	cases := []struct {
		name string
		a, b VectorClock
		want Relation
	}{
		{"相等", VectorClock{"n0": 2, "n1": 1}, VectorClock{"n0": 2, "n1": 1}, RelationEqual},
		{"早于", VectorClock{"n0": 1, "n1": 1}, VectorClock{"n0": 2, "n1": 1}, RelationBefore},
		{"晚于", VectorClock{"n0": 3, "n1": 1}, VectorClock{"n0": 2, "n1": 1}, RelationAfter},
		{"并发", VectorClock{"n0": 2, "n1": 1}, VectorClock{"n0": 1, "n1": 2}, RelationConcurrent},
		{"缺失分量视为零", VectorClock{"n0": 1}, VectorClock{"n0": 1, "n1": 1}, RelationBefore},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.a.Compare(c.b); got != c.want {
				t.Fatalf("Compare = %s, want %s", got, c.want)
			}
		})
	}
}

func TestVectorClockMergeAndClone(t *testing.T) {
	a := VectorClock{"n0": 3, "n1": 1}
	b := VectorClock{"n0": 1, "n2": 7}
	a.Merge(b)
	if a["n0"] != 3 || a["n1"] != 1 || a["n2"] != 7 {
		t.Fatalf("Merge 应逐分量取 max，实际 %v", a)
	}
	clone := a.Clone()
	clone["n0"] = 99
	if a["n0"] != 3 {
		t.Fatal("Clone 必须是深拷贝")
	}
	if !b.HappensBefore(VectorClock{"n0": 1, "n2": 8}) {
		t.Fatal("HappensBefore 判定错误")
	}
	if !(VectorClock{"n0": 2}).Concurrent(VectorClock{"n1": 2}) {
		t.Fatal("Concurrent 判定错误")
	}
	if got := (VectorClock{"n1": 2, "n0": 1}).String(); got != "{n0:1,n1:2}" {
		t.Fatalf("String 应稳定按节点名排序，实际 %s", got)
	}
}

func TestRNGDeterministic(t *testing.T) {
	a, b := NewRNG(20261108), NewRNG(20261108)
	for i := 0; i < 1000; i++ {
		if a.Next() != b.Next() {
			t.Fatalf("同种子第 %d 个值不一致", i)
		}
	}
	if NewRNG(1).Next() == NewRNG(2).Next() {
		t.Fatal("不同种子不应立刻产生相同值")
	}
	for i := 0; i < 100; i++ {
		if v := NewRNG(uint64(i)).Intn(7); v < 0 || v >= 7 {
			t.Fatalf("Intn 越界: %d", v)
		}
	}
}
