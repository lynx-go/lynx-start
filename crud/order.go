package crud

import (
	"fmt"
	"sort"
	"strings"
)

// OrderClause 是一个已解析的排序列：列表达式 + 方向。
type OrderClause struct {
	Expr     string // 白名单映射后的列表达式（含别名，如 "d.created_at"）
	Desc     bool
	RawField string // 原始请求字段名（错误信息用）
}

// ParseOrderBy 解析 AIP-132 排序串（"field desc, other"）到白名单列。
// fields: API 字段名 → SQL 列表达式；未知字段/方向即拒绝（fail-closed）。
func ParseOrderBy(orderBy string, fields map[string]string) ([]OrderClause, error) {
	s := strings.TrimSpace(orderBy)
	if s == "" {
		return nil, nil
	}
	var out []OrderClause
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("order_by: empty clause")
		}
		field, dir := part, "asc"
		if idx := strings.LastIndexAny(part, " \t"); idx >= 0 {
			field, dir = strings.TrimSpace(part[:idx]), strings.TrimSpace(part[idx+1:])
		}
		expr, ok := fields[field]
		if !ok {
			return nil, fmt.Errorf("order_by: unknown field %q", field)
		}
		switch strings.ToLower(dir) {
		case "asc":
		case "desc":
		default:
			return nil, fmt.Errorf("order_by: invalid direction %q", dir)
		}
		out = append(out, OrderClause{Expr: expr, Desc: strings.ToLower(dir) == "desc", RawField: field})
	}
	return out, nil
}

// SortedFields 返回白名单字段名（错误信息/文档生成用，稳定排序）。
func SortedFields(fields map[string]string) []string {
	out := make([]string, 0, len(fields))
	for f := range fields {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}
