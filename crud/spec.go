package crud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// Mode 是一次列表请求经 Spec.Parse 裁决后的执行模式（D11 三模式）。
type Mode uint8

const (
	// ModeBounded 有界全取：小集合（成员/角色/字典），limit 钳制后一次取回。
	ModeBounded Mode = iota
	// ModeOffset 页号跳转：管理台/可跳页场景；O(n) 跳读与并发下漂移由
	// 调用方在 API 文档标注。
	ModeOffset
	// ModeKeyset after 复合游标：feed/日志/收件箱，插入并发下稳定遍历。
	ModeKeyset
)

func (m Mode) String() string {
	switch m {
	case ModeBounded:
		return "bounded"
	case ModeOffset:
		return "offset"
	case ModeKeyset:
		return "keyset"
	default:
		return fmt.Sprintf("Mode(%d)", uint8(m))
	}
}

// Modes 是方法声明的开放模式集合（位掩码），至少声明一个。
type Modes uint8

const (
	ModeSetBounded Modes = 1 << iota
	ModeSetOffset
	ModeSetKeyset
)

func (ms Modes) Allows(m Mode) bool {
	switch m {
	case ModeBounded:
		return ms&ModeSetBounded != 0
	case ModeOffset:
		return ms&ModeSetOffset != 0
	case ModeKeyset:
		return ms&ModeSetKeyset != 0
	default:
		return false
	}
}

func (ms Modes) String() string {
	var out []string
	if ms.Allows(ModeBounded) {
		out = append(out, "bounded")
	}
	if ms.Allows(ModeOffset) {
		out = append(out, "offset")
	}
	if ms.Allows(ModeKeyset) {
		out = append(out, "keyset")
	}
	return strings.Join(out, "|")
}

// Spec 是一个 list 方法声明自己开放的模式与能力（D11：多模式框架）。
// 每个 list 方法一个 Spec 实例——模式、排序白名单、limit 钳制与决胜列
// 都是方法级决策，不是全局默认。
type Spec struct {
	// Modes 本方法开放的分页模式（至少一个）。
	Modes Modes
	// Sortable 排序白名单：API 字段名 → SQL 列表达式（含别名）。
	// 未知字段在 Parse 期 fail-closed。
	Sortable map[string]string
	// Tiebreak 唯一决胜列的 Sortable 键（如 "id"）。offset/keyset 模式
	// 必填：解析后强制追加为排序末段，保证全序稳定（否则翻页可能跳行）。
	Tiebreak string
	// DefaultLimit / MaxLimit 是 ClampLimit 的钳制参数。
	DefaultLimit int32
	MaxLimit     int32
}

// RawParams 是请求侧参数的统一形态：每方法自定义 proto message
// （after_<axis>/offset/limit/order_by 字段）映射到此处——方法面保持
// typed，解析与校验收敛在本包。
type RawParams struct {
	After   string // keyset 游标（编码态）；空 = 第一页
	Offset  int32  // offset 模式跳读行数
	Limit   int32  // <=0 取 DefaultLimit
	OrderBy string // AIP-132 风格白名单排序串
}

// PagePlan 是解析产物的执行面（不携带 filter——过滤是调用方 Query hook
// 的职责），SQL 适配层（如 contrib/bunx 的 ApplyPage）消费本结构。
type PagePlan struct {
	Mode  Mode
	Limit int32
	// Offset 仅 ModeOffset 有效。
	Offset int32
	// Order 已解析排序（含强制追加的决胜列；bounded 无排序请求时可为空）。
	Order []OrderClause
	// CursorValues 仅 ModeKeyset 非首页有效：上一页末行各排序列的编码值，
	// 与 Order 逐列对应。首页为 nil。
	CursorValues []string
}

// Query 携带调用方自定义 filter 类型（D11：自定义 filter 以泛型开放）。
// F 若实现 FilterDigester，keyset 游标会绑定 filter 摘要——翻页中途换滤
// 即自失效；未实现则不绑定（一致性由调用方自行负责，文档需标注）。
type Query[F any] struct {
	PagePlan
	Filter F
}

// FilterDigester 是可选的 filter 绑定钩子：返回 filter 的确定性摘要
// （同 filter 同摘要、不同 filter 不同摘要）。核心提供的 Predicates
// 已实现；自有 filter 类型建议实现以获得游标绑定保护。
type FilterDigester interface {
	FilterDigest() string
}

// ClampLimit 钳制 limit：<=0 取 def；超过 max 取 max（fleetly 式单点）。
func ClampLimit(limit, def, max int32) int32 {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

// HasMore 判定是否可能有下一页（返回行数达到 limit 即认为还有；
// 最终由调用方拿末行值经 Spec.Cursor 编码下一页游标）。
func HasMore(returned, limit int32) bool {
	return limit > 0 && returned >= limit
}

// Parse 解析并校验一次列表请求，产出执行计划（自由函数：Go 方法不能携带
// 类型参数，F 是调用方的 filter 类型）。
//
// 模式裁决（确定性规则）：
//   - After 与 Offset 同时出现：拒绝（语义冲突）；
//   - After 非空 → keyset（须开放）；Offset 非零 → offset（须开放）；
//   - 两者皆空：开放了 keyset 则第一页按 keyset（空游标从头开始），
//     否则开放了 offset 按 offset，否则 bounded。
func Parse[F any](s Spec, raw RawParams, filter F) (Query[F], error) {
	if s.Modes == 0 {
		return Query[F]{}, fmt.Errorf("crud: spec declares no modes")
	}
	if s.DefaultLimit <= 0 || s.MaxLimit <= 0 || s.MaxLimit < s.DefaultLimit {
		return Query[F]{}, fmt.Errorf("crud: spec limits invalid (default=%d max=%d)", s.DefaultLimit, s.MaxLimit)
	}
	if raw.Offset < 0 {
		return Query[F]{}, fmt.Errorf("crud: negative offset %d", raw.Offset)
	}
	if raw.After != "" && raw.Offset != 0 {
		return Query[F]{}, fmt.Errorf("crud: after cursor and offset cannot be combined")
	}

	mode, err := s.resolveMode(raw.After != "", raw.Offset != 0)
	if err != nil {
		return Query[F]{}, err
	}

	clauses, err := ParseOrderBy(raw.OrderBy, s.Sortable)
	if err != nil {
		return Query[F]{}, err
	}

	plan := PagePlan{Mode: mode, Limit: ClampLimit(raw.Limit, s.DefaultLimit, s.MaxLimit), Offset: raw.Offset}
	if mode != ModeBounded {
		if s.Tiebreak == "" {
			return Query[F]{}, fmt.Errorf("crud: spec requires Tiebreak for %s mode", mode)
		}
		tbExpr, ok := s.Sortable[s.Tiebreak]
		if !ok {
			return Query[F]{}, fmt.Errorf("crud: spec Tiebreak %q is not in Sortable", s.Tiebreak)
		}
		clauses = appendTiebreak(clauses, s.Tiebreak, tbExpr)
	}
	plan.Order = clauses

	q := Query[F]{PagePlan: plan, Filter: filter}
	if mode == ModeKeyset && raw.After != "" {
		tok, err := decodeKeysetToken(raw.After)
		if err != nil {
			return Query[F]{}, fmt.Errorf("crud: invalid after cursor: %w", err)
		}
		if tok.Order != orderDigest(clauses) {
			return Query[F]{}, fmt.Errorf("crud: after cursor was issued for a different order_by")
		}
		if want := filterDigestOf(filter); tok.Filter != want {
			return Query[F]{}, fmt.Errorf("crud: after cursor was issued for a different filter")
		}
		if len(tok.Values) != len(clauses) {
			return Query[F]{}, fmt.Errorf("crud: after cursor has %d values, order has %d columns", len(tok.Values), len(clauses))
		}
		q.CursorValues = tok.Values
	}
	return q, nil
}

// Cursor 把上一页末行的排序键值编码为下一页游标（encode-not-sign：
// 编码达成不透明性，不做 HMAC——列表非机密，摘要绑定换序/换滤自失效）。
// 仅 keyset 模式有效。
func Cursor[F any](q Query[F], lastRowValues []string) (string, error) {
	if q.Mode != ModeKeyset {
		return "", fmt.Errorf("crud: cursor is only available in keyset mode, got %s", q.Mode)
	}
	if len(lastRowValues) != len(q.Order) {
		return "", fmt.Errorf("crud: cursor needs %d values (one per order column incl. tiebreak), got %d", len(q.Order), len(lastRowValues))
	}
	tok := keysetToken{Order: orderDigest(q.Order), Filter: filterDigestOf(q.Filter), Values: lastRowValues}
	blob, err := json.Marshal(tok)
	if err != nil {
		return "", fmt.Errorf("crud: encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func (s Spec) resolveMode(afterSet, offsetSet bool) (Mode, error) {
	var want Mode
	switch {
	case afterSet:
		want = ModeKeyset
	case offsetSet:
		want = ModeOffset
	case s.Modes.Allows(ModeKeyset):
		want = ModeKeyset
	case s.Modes.Allows(ModeOffset):
		want = ModeOffset
	default:
		want = ModeBounded
	}
	if !s.Modes.Allows(want) {
		return 0, fmt.Errorf("crud: %s mode is not enabled for this method (enabled: %s)", want, s.Modes)
	}
	return want, nil
}

// appendTiebreak 追加决胜列为排序末段；排序已含该列表达式则不重复。
func appendTiebreak(clauses []OrderClause, field, expr string) []OrderClause {
	for _, c := range clauses {
		if c.Expr == expr {
			return clauses
		}
	}
	return append(clauses, OrderClause{Expr: expr, Desc: false, RawField: field})
}

// keysetToken 是游标的载荷形态（JSON+base64url 编码）。
type keysetToken struct {
	Order  string   `json:"o"` // order digest（绑定排序）
	Filter string   `json:"f"` // filter digest（可空 = 未绑定）
	Values []string `json:"v"` // 末行排序键值向量
}

func decodeKeysetToken(raw string) (keysetToken, error) {
	var tok keysetToken
	blob, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return tok, err
	}
	if err := json.Unmarshal(blob, &tok); err != nil {
		return tok, err
	}
	if len(tok.Values) == 0 {
		return tok, fmt.Errorf("empty values")
	}
	return tok, nil
}

// orderDigest 返回排序子句的确定性摘要（expr+方向，逐段拼接）。
func orderDigest(clauses []OrderClause) string {
	var b strings.Builder
	for i, c := range clauses {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(c.Expr)
		if c.Desc {
			b.WriteString(" desc")
		}
	}
	return b.String()
}

// filterDigestOf 提取 filter 摘要；未实现 FilterDigester 返回空（不绑定）。
func filterDigestOf[F any](filter F) string {
	if d, ok := any(filter).(FilterDigester); ok {
		return d.FilterDigest()
	}
	return ""
}
