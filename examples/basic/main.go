// Command basic 演示如何把 ConfluxLab 当作库使用。
package main

import (
	"fmt"

	"github.com/gitstq/ConfluxLab"
)

func main() {
	// 两个副本在分区期间各自演化。
	left, right := confluxlab.NewORSet(), confluxlab.NewORSet()
	left.Add("deploy", "n0:1")
	_ = right.Merge(left.Clone())
	left.Remove("deploy")
	right.Add("deploy", "n1:9")

	// 网络恢复后双向合并。
	_ = left.Merge(right.Clone())
	_ = right.Merge(left.Clone())
	fmt.Println("合并后可见元素:", left.Elements())
	fmt.Println("两副本是否一致:", fmt.Sprint(left.Snapshot()) == fmt.Sprint(right.Snapshot()))

	// 用确定性沙盘做回归测试。
	cfg := confluxlab.DefaultConfig()
	cfg.Rounds = 200
	report, err := confluxlab.RunSandbox(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(report.Summary())
}
