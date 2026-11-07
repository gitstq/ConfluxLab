package confluxlab

import (
	"fmt"
	"reflect"
)

// Op 描述一次施加在某个副本上的本地操作。
// Stamp 为发起副本的 Lamport 逻辑时戳，Node 为副本 ID，二者共同保证标识唯一。
type Op struct {
	Type  string // inc / dec / set / add / remove
	Arg   string // 目标值或元素
	Node  string // 发起副本
	Stamp uint64 // 逻辑时戳
}

// String 返回操作的可读表示。
func (o Op) String() string {
	if o.Arg == "" {
		return fmt.Sprintf("%s@%s#%d", o.Type, o.Node, o.Stamp)
	}
	return fmt.Sprintf("%s(%s)@%s#%d", o.Type, o.Arg, o.Node, o.Stamp)
}

// State 是所有状态型 CRDT 的统一契约。
// Merge 必须是幂等、可交换、可结合的最小上界（join-semilattice）运算，
// 这是所有副本最终收敛的数学前提。
type State interface {
	Kind() string
	Clone() State
	Merge(State) error
	Snapshot() any
	Apply(Op) error
}

// AllKinds 返回内置的全部 CRDT 类型。
func AllKinds() []string {
	return []string{"gcounter", "pncounter", "lwwregister", "mvregister", "orset"}
}

// NewState 按类型名创建空状态。
func NewState(kind string) (State, error) {
	switch kind {
	case "gcounter":
		return NewGCounter(), nil
	case "pncounter":
		return NewPNCounter(), nil
	case "lwwregister":
		return NewLWWRegister(), nil
	case "mvregister":
		return NewMVRegister(), nil
	case "orset":
		return NewORSet(), nil
	default:
		return nil, fmt.Errorf("未知的 CRDT 类型: %s", kind)
	}
}

func typeError(want string, got State) error {
	return fmt.Errorf("类型不匹配: 期望 %s，实际 %s", want, got.Kind())
}

func errUnsupported(op Op, kind string) error {
	return fmt.Errorf("%s 不支持操作 %s", kind, op.Type)
}

// snapshotsEqual 用反射深度比较两个快照是否逐位一致。
func snapshotsEqual(a, b any) bool { return reflect.DeepEqual(a, b) }

// snapshotString 输出快照的稳定文本，用于报告与调试。
func snapshotString(v any) string { return fmt.Sprintf("%v", v) }
