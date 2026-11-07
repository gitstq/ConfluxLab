package confluxlab

import "testing"

func TestSandboxReportHelpers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rounds = 60
	report, err := RunSandbox(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Summary(); got == "" {
		t.Fatal("Summary 不应为空")
	}
	rendered := RenderTimeline(report.Kinds[0])
	if rendered == "" {
		t.Fatal("时间线渲染不应为空")
	}
	for _, k := range report.Kinds {
		if !k.Ok() {
			t.Errorf("%s 报告异常: %v", k.Kind, k.Violations)
		}
	}
}

func TestNewStateAllKinds(t *testing.T) {
	for _, kind := range AllKinds() {
		st, err := NewState(kind)
		if err != nil {
			t.Fatal(err)
		}
		if st.Kind() != kind {
			t.Fatalf("Kind 不匹配: %s vs %s", st.Kind(), kind)
		}
		if st.Clone().Kind() != kind {
			t.Fatalf("%s Clone 后 Kind 丢失", kind)
		}
	}
}
