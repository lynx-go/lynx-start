package goosekit

import (
	"testing"
)

func TestTableName(t *testing.T) {
	if got := TableName("identity"); got != "goose_identity_db" {
		t.Errorf("TableName(identity) = %q", got)
	}
	if got := TableName("my-module"); got != "goose_my_module_db" {
		t.Errorf("TableName(my-module) = %q", got)
	}
}

// 防注入负路径：不安全的模块名派生出的表名必须被拒（provider 构造期）。
func TestUnsafeTableNameRejected(t *testing.T) {
	for _, bad := range []string{"Demo", `x"; drop table users; --`, "mod ule"} {
		table := TableName(bad)
		if tableNameRe.MatchString(table) {
			t.Errorf("TableName(%q) = %q should be rejected", bad, table)
		}
	}
}
