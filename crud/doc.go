// Package crud 是列表与排序的公共模式基元（ADR-0009 D11 分页定典）：
// 分页参数/收尾、时间游标、排序白名单解析——ORM 中立，任何数据访问栈
// 可直接消费（bun 的 SQL 编译见 contrib/bunx）。
//
// # 分页范式（v2 方向，dogfood 首循环落地）
//
// crud 定位为多模式列表框架，每个 list 方法声明开放的模式：
//
//   - bounded：有界全取（小集合：成员/角色/字典），ClampLimit 钳制；
//   - offset：页号跳转（管理台/小中集合），O(n) 跳读与并发漂移需文档标注；
//   - keyset：after 复合游标（feed/日志/收件箱），插入并发下稳定遍历。
//
// 自定义 order_by 是一等能力：字段白名单 + 强制唯一 tiebreak 保证全序；
// keyset 游标编码「排序键值向量 + order 摘要 + filter 摘要」（encode-
// not-sign：不透明性用编码达成，不做 HMAC 签名——列表非机密，篡改只害
// 篡改者；摘要令换序/换滤自失效）。自定义 filter 以 Params[F] 泛型携带
// 每方法自有类型；核心只提供最小 typed 谓词集，字符串 DSL 留在应用层。
//
// 当前 API（ListParams/ParseListParams/FinalizeOffsetPage/TimeCursor +
// HMAC 签名 token）为 v0.4 既有形态，v0.5.0 保持编译兼容；签名 token
// 机制将从核心范式除名，随模块换装 crud v2 退场。crud 是库级辅助件而
// 非契约强制——自带分页实现（如裸 keyset）不违范式。
package crud
