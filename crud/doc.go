// Package crud 是列表与排序的公共模式基元（ADR-0009 D11 分页定典）——
// ORM 中立，任何数据访问栈可直接消费（bun 的 SQL 编译见 contrib/bunx
// 的 ApplyPage/ApplyOrder）。
//
// # v2：多模式分页框架（Spec / Parse / Cursor / ApplyPage）
//
// 每个 list 方法声明一个 Spec（开放的模式、排序白名单、limit 钳制、
// 唯一决胜列），请求侧 proto 字段映射到 RawParams：
//
//	spec := crud.Spec{
//	    Modes:        crud.ModeSetKeyset, // 或 | ModeSetOffset / ModeSetBounded
//	    Sortable:     map[string]string{"createdAt": "d.created_at", "id": "d.id"},
//	    Tiebreak:     "id",               // keyset/offset 必填，强制追加为排序末段
//	    DefaultLimit: 50, MaxLimit: 200,
//	}
//	q, err := crud.Parse[any](spec, crud.RawParams{After: after, OrderBy: orderBy}, filter)
//	// q.Mode / q.Limit / q.Order（含决胜列）/ q.CursorValues / q.Filter
//	// …执行查询（bunx.ApplyPage）；返回行数==limit 且拿得到末行值时：
//	next, err := crud.Cursor(q, lastRowValues)
//
// 三模式：bounded（有界全取，ClampLimit 钳制）/ offset（页跳转，调用方
// 文档标注 O(n) 与并发漂移）/ keyset（after 复合游标，插入并发稳定）。
// 模式裁决：After 非空→keyset；Offset 非零→offset；皆空时按
// keyset>offset>bounded 的已开放优先级取第一页。
//
// 自定义 order_by 是一等能力：白名单解析（ParseOrderBy）+ 决胜列强制 +
// keyset 游标编码「排序键值向量 + order 摘要 + filter 摘要」——
// encode-not-sign：不透明性用编码达成，不做 HMAC（列表非机密，篡改只害
// 篡改者）；摘要令换序/换滤自失效。filter 以 Parse 的类型参数 F 开放：
// 核心提供最小 typed 谓词集 Predicates（等值/IN/区间，AND 组合）作
// 现成实现，F 实现 FilterDigester 即获得游标绑定保护。字符串 DSL（富
// 查询 AST）属应用层。
//
// # v1（已 Deprecated）
//
// ListParams/ParseListParams/FinalizeOffsetPage/SliceOffsetPage/
// InitPageTokenSigning/TimeCursor 等 AIP-158 签名 token 形态保持编译
// 兼容，随模块换装 v2 退场。crud 是库级辅助件而非契约强制——自带分页
// 实现（如裸 keyset）不违范式。
package crud
