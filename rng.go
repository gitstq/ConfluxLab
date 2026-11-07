// Package confluxlab 是一个零依赖的状态型 CRDT 工具箱与随机并发收敛性验证沙盘。
//
// 与"再实现一遍 CRDT"不同，ConfluxLab 的核心目标是把「收敛性」变成可测量、
// 可回归测试的指标：通过确定性随机调度，把多副本的本地操作、网络分区、
// 消息乱序与重复投递交错执行，最终自动验证所有副本是否收敛到同一状态，
// 并检查 CRDT 的交换律、结合律、幂等律以及各自特有的不变量。
package confluxlab

// RNG 是零依赖的确定性伪随机数发生器（SplitMix64）。
// 同一 seed 必然产生逐位一致的序列，因此任何一次调度都可以精确回放。
type RNG struct {
	state uint64
}

// NewRNG 以给定种子创建发生器。
func NewRNG(seed uint64) *RNG { return &RNG{state: seed} }

// Next 返回下一个 64 位随机值。
func (r *RNG) Next() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Intn 返回 [0, n) 区间内的整数；n <= 0 时返回 0。
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.Next() % uint64(n))
}

// Uint64n 返回 [0, n) 区间内的无符号整数。
func (r *RNG) Uint64n(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return r.Next() % n
}

// Bool 返回一个随机布尔值。
func (r *RNG) Bool() bool { return r.Next()&1 == 1 }

// Chance 以概率 p（0.0~1.0）返回 true。
func (r *RNG) Chance(p float64) bool {
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	return float64(r.Next()%1_000_000)/1_000_000 < p
}
