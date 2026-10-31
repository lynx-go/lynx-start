package crud

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

// Op 是最小 typed 谓词集的比较算子（D11：核心只提供等值/IN/区间+组合的
// 通用件，字符串 DSL 与富查询 AST 属应用层——torchwood pkg/query 形态）。
type Op uint8

const (
	OpEqual Op = iota + 1
	OpNotEqual
	OpIn
	OpGreater
	OpGreaterOrEqual
	OpLess
	OpLessOrEqual
)

func (o Op) String() string {
	switch o {
	case OpEqual:
		return "="
	case OpNotEqual:
		return "!="
	case OpIn:
		return "in"
	case OpGreater:
		return ">"
	case OpGreaterOrEqual:
		return ">="
	case OpLess:
		return "<"
	case OpLessOrEqual:
		return "<="
	default:
		return strconv.Itoa(int(o))
	}
}

// Predicate 是单个过滤条件：列（白名单键）× 算子 × 值集。
type Predicate struct {
	Column string
	Op     Op
	Values []string
}

// Predicates 是核心提供的最小 typed 过滤集，AND 组合——可直接用作
// Query[F] 的 F，也可以只是参考形态（自有 filter 类型携带任意结构）。
// 列合法性由调用方按方法白名单校验（与排序白名单同理）。
type Predicates struct {
	Conds []Predicate
}

// FilterDigest 实现 FilterDigester：对条件集合做规范化（列+算子排序）
// 后取摘要——条件顺序无关，同滤同摘要。
func (p Predicates) FilterDigest() string {
	if len(p.Conds) == 0 {
		return ""
	}
	parts := make([]string, 0, len(p.Conds))
	for _, c := range p.Conds {
		vals := append([]string(nil), c.Values...)
		sort.Strings(vals)
		parts = append(parts, c.Column+" "+c.Op.String()+" "+strings.Join(vals, ","))
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, ";")))
	return hex.EncodeToString(sum[:])
}
