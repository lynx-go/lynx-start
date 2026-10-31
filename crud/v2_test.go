package crud

import (
	"strings"
	"testing"
)

func v2Spec() Spec {
	return Spec{
		Modes:        ModeSetBounded | ModeSetOffset | ModeSetKeyset,
		Sortable:     map[string]string{"createdAt": "d.created_at", "state": "d.state", "id": "d.id"},
		Tiebreak:     "id",
		DefaultLimit: 50,
		MaxLimit:     200,
	}
}

func TestClampLimit(t *testing.T) {
	if got := ClampLimit(0, 50, 200); got != 50 {
		t.Errorf("zero → default: got %d", got)
	}
	if got := ClampLimit(-5, 50, 200); got != 50 {
		t.Errorf("negative → default: got %d", got)
	}
	if got := ClampLimit(999, 50, 200); got != 200 {
		t.Errorf("over → max: got %d", got)
	}
	if got := ClampLimit(20, 50, 200); got != 20 {
		t.Errorf("in-range passthrough: got %d", got)
	}
}

// TestModeResolutionMatrix 三模式 × 请求形态的裁决矩阵。
func TestModeResolutionMatrix(t *testing.T) {
	cases := []struct {
		name    string
		modes   Modes
		raw     RawParams
		want    Mode
		wantErr bool
	}{
		{"bounded only / plain", ModeSetBounded, RawParams{}, ModeBounded, false},
		{"bounded only / after rejected", ModeSetBounded, RawParams{After: "x"}, 0, true},
		{"bounded only / offset rejected", ModeSetBounded, RawParams{Offset: 10}, 0, true},
		{"keyset only / first page", ModeSetKeyset, RawParams{}, ModeKeyset, false},
		{"keyset only / with after", ModeSetKeyset, RawParams{After: "x"}, 0, true}, // "x" 非法 token → err（模式本身允许）
		{"keyset only / offset rejected", ModeSetKeyset, RawParams{Offset: 10}, 0, true},
		{"offset only / first page", ModeSetOffset, RawParams{}, ModeOffset, false},
		{"offset only / with offset", ModeSetOffset, RawParams{Offset: 40}, ModeOffset, false},
		{"offset only / after rejected", ModeSetOffset, RawParams{After: "x"}, 0, true},
		{"keyset+offset / first page prefers keyset", ModeSetOffset | ModeSetKeyset, RawParams{}, ModeKeyset, false},
		{"keyset+offset / explicit offset", ModeSetOffset | ModeSetKeyset, RawParams{Offset: 40}, ModeOffset, false},
		{"after + offset combined rejected", ModeSetOffset | ModeSetKeyset, RawParams{After: "x", Offset: 5}, 0, true},
	}
	for _, tc := range cases {
		spec := v2Spec()
		spec.Modes = tc.modes
		q, err := Parse[any](spec, tc.raw, nil)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%s: want error, got mode=%s", tc.name, q.Mode)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if q.Mode != tc.want {
			t.Errorf("%s: mode=%s want %s", tc.name, q.Mode, tc.want)
		}
	}
}

func TestParseValidation(t *testing.T) {
	// spec 缺模式 / limit 配置非法 / 负 offset / 未知排序字段 / 坏方向。
	if _, err := Parse[any](Spec{Modes: 0}, RawParams{}, nil); err == nil {
		t.Error("no modes: want error")
	}
	bad := v2Spec()
	bad.MaxLimit = 10
	if _, err := Parse[any](bad, RawParams{}, nil); err == nil {
		t.Error("max < default: want error")
	}
	if _, err := Parse[any](v2Spec(), RawParams{Offset: -1}, nil); err == nil {
		t.Error("negative offset: want error")
	}
	if _, err := Parse[any](v2Spec(), RawParams{OrderBy: "hacker"}, nil); err == nil {
		t.Error("unknown order field: want error")
	}
	if _, err := Parse[any](v2Spec(), RawParams{OrderBy: "id sideways"}, nil); err == nil {
		t.Error("bad direction: want error")
	}
	// keyset/offset 模式缺 Tiebreak / Tiebreak 不在白名单。
	noTb := v2Spec()
	noTb.Tiebreak = ""
	if _, err := Parse[any](noTb, RawParams{}, nil); err == nil {
		t.Error("missing tiebreak: want error")
	}
	badTb := v2Spec()
	badTb.Tiebreak = "nope"
	if _, err := Parse[any](badTb, RawParams{}, nil); err == nil {
		t.Error("tiebreak not sortable: want error")
	}
	// bounded-only 不需要 tiebreak。
	bounded := Spec{Modes: ModeSetBounded, Sortable: v2Spec().Sortable, DefaultLimit: 10, MaxLimit: 20}
	if q, err := Parse[any](bounded, RawParams{}, nil); err != nil || q.Mode != ModeBounded {
		t.Errorf("bounded without tiebreak: q=%+v err=%v", q, err)
	}
}

func TestTiebreakEnforced(t *testing.T) {
	// 无排序请求：决胜列强制追加为唯一排序。
	q, err := Parse[any](v2Spec(), RawParams{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Order) != 1 || q.Order[0].Expr != "d.id" || q.Order[0].Desc {
		t.Errorf("forced tiebreak missing: %+v", q.Order)
	}
	// 显式排序：决胜列追加为末段。
	q, err = Parse[any](v2Spec(), RawParams{OrderBy: "createdAt desc"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Order) != 2 || q.Order[0].Expr != "d.created_at" || !q.Order[0].Desc || q.Order[1].Expr != "d.id" {
		t.Errorf("tiebreak not appended: %+v", q.Order)
	}
	// 排序已含决胜列：不重复。
	q, err = Parse[any](v2Spec(), RawParams{OrderBy: "createdAt, id"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Order) != 2 {
		t.Errorf("tiebreak duplicated: %+v", q.Order)
	}
}

func TestKeysetCursorRoundtrip(t *testing.T) {
	filter := Predicates{Conds: []Predicate{{Column: "state", Op: OpEqual, Values: []string{"active"}}}}
	spec := v2Spec()

	// 第一页（空游标）。
	q1, err := Parse[any](spec, RawParams{OrderBy: "createdAt desc"}, filter)
	if err != nil {
		t.Fatal(err)
	}
	if q1.Mode != ModeKeyset || len(q1.CursorValues) != 0 {
		t.Fatalf("first page: %+v", q1.PagePlan)
	}
	// 末行值 → 下一页游标。
	tok, err := Cursor(q1, []string{"2026-10-31T00:00:00Z", "row-42"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tok, "row-42|") { // 编码态，肉眼不可读即达标
		t.Errorf("cursor not opaque enough: %s", tok)
	}
	// 第二页：同序同滤 → 游标值还原。
	q2, err := Parse[any](spec, RawParams{After: tok, OrderBy: "createdAt desc"}, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(q2.CursorValues) != 2 || q2.CursorValues[0] != "2026-10-31T00:00:00Z" || q2.CursorValues[1] != "row-42" {
		t.Errorf("cursor values not restored: %v", q2.CursorValues)
	}
	if !HasMore(50, q2.Limit) || HasMore(49, q2.Limit) {
		t.Error("HasMore semantics broken")
	}

	// 换序 → 拒。
	if _, err := Parse[any](spec, RawParams{After: tok, OrderBy: "createdAt"}, filter); err == nil {
		t.Error("order change must invalidate cursor")
	}
	// 换滤（Predicates 带 digest 绑定）→ 拒。
	other := Predicates{Conds: []Predicate{{Column: "state", Op: OpEqual, Values: []string{"closed"}}}}
	if _, err := Parse[any](spec, RawParams{After: tok, OrderBy: "createdAt desc"}, other); err == nil {
		t.Error("filter change must invalidate cursor (digested filter)")
	}
	// 游标值数量不匹配 → Cursor/Parse 双侧拒绝。
	if _, err := Cursor(q1, []string{"only-one"}); err == nil {
		t.Error("Cursor with wrong value count: want error")
	}
	if _, err := Parse[any](spec, RawParams{After: "not-a-token"}, filter); err == nil {
		t.Error("malformed cursor: want error")
	}
}

// TestCursorFilterUnbound：filter 未实现 FilterDigester 时不绑定
// （换滤不拒——一致性由调用方负责，文档已标注）。
func TestCursorFilterUnbound(t *testing.T) {
	spec := v2Spec()
	type plainFilter struct{ Status string }
	q1, err := Parse[any](spec, RawParams{OrderBy: "id"}, plainFilter{"a"})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := Cursor(q1, []string{"row-42"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse[any](spec, RawParams{After: tok, OrderBy: "id"}, plainFilter{"b"}); err != nil {
		t.Errorf("unbound filter must not reject: %v", err)
	}
	// 换序仍然拒绝（order 摘要始终绑定）。
	if _, err := Parse[any](spec, RawParams{After: tok, OrderBy: "createdAt"}, plainFilter{"a"}); err == nil {
		t.Error("order binding must always hold")
	}
}

func TestPredicatesDigest(t *testing.T) {
	a := Predicates{Conds: []Predicate{
		{Column: "state", Op: OpEqual, Values: []string{"active"}},
		{Column: "created", Op: OpGreaterOrEqual, Values: []string{"2026-01-01"}},
	}}
	b := Predicates{Conds: []Predicate{ // 条件顺序不同、值序不同 → 同滤
		{Column: "created", Op: OpGreaterOrEqual, Values: []string{"2026-01-01"}},
		{Column: "state", Op: OpEqual, Values: []string{"active"}},
	}}
	if a.FilterDigest() == "" || a.FilterDigest() != b.FilterDigest() {
		t.Errorf("digest must be canonical: %q vs %q", a.FilterDigest(), b.FilterDigest())
	}
	c := Predicates{Conds: []Predicate{{Column: "state", Op: OpEqual, Values: []string{"closed"}}}}
	if a.FilterDigest() == c.FilterDigest() {
		t.Error("different filters must differ")
	}
	if (Predicates{}).FilterDigest() != "" {
		t.Error("empty predicates digest must be empty")
	}
}
