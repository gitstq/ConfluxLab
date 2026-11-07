package confluxlab

import (
	"encoding/json"
	"testing"
)

func TestCheckLawsAllKinds(t *testing.T) {
	results, err := CheckLaws(20261108, 120, AllKinds())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(AllKinds()) {
		t.Fatalf("应返回 %d 个类型的结果，实际 %d", len(AllKinds()), len(results))
	}
	for _, r := range results {
		if !r.Ok() {
			t.Errorf("%s 代数律失败: %v", r.Kind, r.Violations)
		}
		if r.Checked != 360 {
			t.Errorf("%s 检查次数应为 120*3=360，实际 %d", r.Kind, r.Checked)
		}
	}
}

func TestCheckLawsDifferentSeeds(t *testing.T) {
	for _, seed := range []uint64{0, 1, 42, 999983, 20261108} {
		results, err := CheckLaws(seed, 40, AllKinds())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range results {
			if !r.Ok() {
				t.Errorf("seed=%d %s 失败: %v", seed, r.Kind, r.Violations)
			}
		}
	}
}

func TestSandboxConverges(t *testing.T) {
	for _, seed := range []uint64{1, 42, 777, 20261108} {
		cfg := DefaultConfig()
		cfg.Seed, cfg.Rounds, cfg.Replicas = seed, 300, 4
		report, err := RunSandbox(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !report.Ok() {
			t.Errorf("seed=%d 未收敛: %v", seed, report.Violations)
		}
		for _, k := range report.Kinds {
			if k.LocalOps == 0 || k.Merges == 0 {
				t.Errorf("seed=%d %s 调度未产生足够事件", seed, k.Kind)
			}
		}
	}
}

func TestSandboxDeterministicReplay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Seed, cfg.Rounds = 20261108, 250
	first, err := RunSandbox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunSandbox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("同一种子两次运行必须逐位一致（可精确回放）")
	}
	cfg.Seed = 20261109
	third, err := RunSandbox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := json.Marshal(third)
	if string(a) == string(c) {
		t.Fatal("不同种子应产生不同调度")
	}
}

func TestSandboxReplicasAndKinds(t *testing.T) {
	for _, replicas := range []int{2, 3, 6} {
		cfg := DefaultConfig()
		cfg.Replicas, cfg.Rounds, cfg.Seed = replicas, 150, 12345
		cfg.Kinds = []string{"orset", "mvregister"}
		report, err := RunSandbox(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Kinds) != 2 {
			t.Fatal("应只运行指定的 2 个类型")
		}
		if !report.Ok() {
			t.Errorf("replicas=%d 未收敛: %v", replicas, report.Violations)
		}
	}
}

func TestSandboxJSONRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rounds = 80
	report, err := RunSandbox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var decoded SandboxReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Seed != report.Seed || len(decoded.Kinds) != len(report.Kinds) {
		t.Fatal("JSON 往返丢失字段")
	}
	if decoded.Summary() != report.Summary() {
		t.Fatal("JSON 往返后摘要不一致")
	}
}
