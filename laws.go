package confluxlab

import "fmt"

// LawResult 记录某一 CRDT 类型的三条代数律验证结果。
type LawResult struct {
	Kind       string
	Checked    int
	Violations []string
}

// Ok 报告该类型是否全部通过。
func (r LawResult) Ok() bool { return len(r.Violations) == 0 }

// randomState 生成一个经过若干随机本地操作的状态，用于代数律验证。
// counter 是本轮 a/b/c 三个状态共享的全局逻辑时戳源：CRDT 要求每次写入的
// 标识（时戳+节点，或 tag）全局唯一，若各状态各自从 1 开始编号就会人为制造
// 标识冲突，从而把"生成器不合法"误判为"代数律不成立"。
// 对 MVRegister 额外注入来自不同节点的并发时钟，以覆盖多值冲突分支。
func randomState(kind string, rng *RNG, nodes []string, steps int, counter *uint64) (State, error) {
	st, err := NewState(kind)
	if err != nil {
		return nil, err
	}
	universe := []string{"alpha", "beta", "gamma", "delta"}
	for i := 0; i < steps; i++ {
		node := nodes[rng.Intn(len(nodes))]
		arg := universe[rng.Intn(len(universe))]
		*counter++
		stamp := *counter
		if kind == "mvregister" {
			clock := NewVectorClock()
			clock[node] = stamp
			other := nodes[rng.Intn(len(nodes))]
			if other != node {
				clock[other] = uint64(1 + rng.Intn(3))
			}
			st.(*MVRegister).SetWithClock(arg, clock)
			continue
		}
		op := randomOp(kind, rng, node, stamp, universe)
		if err := st.Apply(op); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// randomOp 依据类型生成一次合法的本地操作。
func randomOp(kind string, rng *RNG, node string, stamp uint64, universe []string) Op {
	arg := universe[rng.Intn(len(universe))]
	switch kind {
	case "gcounter":
		return Op{Type: "inc", Node: node, Stamp: stamp}
	case "pncounter":
		t := "inc"
		if rng.Bool() {
			t = "dec"
		}
		return Op{Type: t, Node: node, Stamp: stamp}
	case "lwwregister", "mvregister":
		return Op{Type: "set", Arg: arg, Node: node, Stamp: stamp}
	case "orset":
		t := "add"
		if rng.Chance(0.35) {
			t = "remove"
		}
		return Op{Type: t, Arg: arg, Node: node, Stamp: stamp}
	default:
		return Op{Type: "inc", Node: node, Stamp: stamp}
	}
}

// CheckLaws 验证每种 CRDT 的交换律、结合律与幂等律。
// 这三条定律是 join-semilattice 的定义性质，也是收敛性的充分条件；
// 任意一条被破坏，副本就无法保证最终一致。
func CheckLaws(seed uint64, rounds int, kinds []string) ([]LawResult, error) {
	if rounds <= 0 {
		rounds = 100
	}
	if len(kinds) == 0 {
		kinds = AllKinds()
	}
	nodes := []string{"n0", "n1", "n2", "n3"}
	results := make([]LawResult, 0, len(kinds))
	for idx, kind := range kinds {
		if _, err := NewState(kind); err != nil {
			return nil, err
		}
		rng := NewRNG(seed + uint64(idx)*0x9E3779B97F4A7C15)
		res := LawResult{Kind: kind, Violations: []string{}}
		var stampCounter uint64 // 跨轮次持续递增，保证所有生成的写入标识唯一
		for round := 0; round < rounds; round++ {
			a, err := randomState(kind, rng, nodes, 2+rng.Intn(6), &stampCounter)
			if err != nil {
				return nil, err
			}
			b, err := randomState(kind, rng, nodes, 2+rng.Intn(6), &stampCounter)
			if err != nil {
				return nil, err
			}
			c, err := randomState(kind, rng, nodes, 2+rng.Intn(6), &stampCounter)
			if err != nil {
				return nil, err
			}
			res.Checked += 3

			// 交换律: a ⊔ b == b ⊔ a
			ab, ba := a.Clone(), b.Clone()
			if err := ab.Merge(b.Clone()); err != nil {
				return nil, err
			}
			if err := ba.Merge(a.Clone()); err != nil {
				return nil, err
			}
			if !snapshotsEqual(ab.Snapshot(), ba.Snapshot()) {
				res.Violations = append(res.Violations,
					fmt.Sprintf("第 %d 轮交换律失败: %v != %v", round, ab.Snapshot(), ba.Snapshot()))
			}

			// 结合律: (a ⊔ b) ⊔ c == a ⊔ (b ⊔ c)
			left := a.Clone()
			if err := left.Merge(b.Clone()); err != nil {
				return nil, err
			}
			if err := left.Merge(c.Clone()); err != nil {
				return nil, err
			}
			bc := b.Clone()
			if err := bc.Merge(c.Clone()); err != nil {
				return nil, err
			}
			right := a.Clone()
			if err := right.Merge(bc); err != nil {
				return nil, err
			}
			if !snapshotsEqual(left.Snapshot(), right.Snapshot()) {
				res.Violations = append(res.Violations,
					fmt.Sprintf("第 %d 轮结合律失败", round))
			}

			// 幂等律: a ⊔ a == a
			idem := a.Clone()
			before := a.Snapshot()
			if err := idem.Merge(a.Clone()); err != nil {
				return nil, err
			}
			if !snapshotsEqual(idem.Snapshot(), before) {
				res.Violations = append(res.Violations,
					fmt.Sprintf("第 %d 轮幂等律失败", round))
			}
		}
		results = append(results, res)
	}
	return results, nil
}
