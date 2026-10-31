# 升级指南（UPGRADING）

## kernel v0.4.0（github.com/lynx-go/lynx-start/kernel）→ lynx-start v0.5.0（github.com/lynx-go/lynx-start）

v0.5.0 是一次破坏性重排（0.x 阶段 minor 破坏承诺内）：仓库转型为
web 开发基础库，kernel 升为仓库根 module（ADR-0009）。

### 1. import 路径

| 旧（kernel v0.4.0） | 新（v0.5.0） |
|---|---|
| `github.com/lynx-go/lynx-start/kernel`（根包） | `github.com/lynx-go/lynx-start` |
| `github.com/lynx-go/lynx-start/kernel/uow` | `github.com/lynx-go/lynx-start/uow` |
| `github.com/lynx-go/lynx-start/kernel/crud` | `github.com/lynx-go/lynx-start/crud`（新增 OrderClause/ParseOrderBy/SortedFields） |
| `github.com/lynx-go/lynx-start/kernel/bunx` | `github.com/lynx-go/lynx-start/contrib/bunx` |
| `github.com/lynx-go/lynx-start/kernel/uow/buntx` | `github.com/lynx-go/lynx-start/contrib/bunx/buntx` |
| `github.com/lynx-go/lynx-start/kernel/testutil` | `github.com/lynx-go/lynx-start/contrib/bunx/testutil` |
| 其余 `…/kernel/<子包>` | `…/lynx-start/<子包>`（同名平移） |

包名与限定符**零变化**（根包仍叫 `kernel`，`kernel.Deps`/`kernel.Module`
照写——goimports 可能猜错包名，显式别名 `kernel "github.com/lynx-go/lynx-start"` 最稳）。

### 2. 实现函数换包（根包纯契约化）

| 旧 | 新 |
|---|---|
| `kernel.ValidateModules` | `compose.ValidateModules` |
| `kernel.AssembleAll` | `compose.AssembleAll` |
| `kernel.PolicyFiles` / `kernel.SortModules` / `kernel.Descriptors` | `compose.` 同名 |
| `kernel.DefaultChain` | `chain.DefaultChain` |

### 3. Deps.DB 类型变化（ORM 无关化）

`kernel.Deps.DB` 由 `*bun.DB` 改为 `any`。bun 应用的装配处加一次类型化
出口；仓储构造调用点：

```go
// 旧
repo := infraidentity.NewRepository(deps.DB)
// 新
repo := infraidentity.NewRepository(bunx.MustDB(deps)) // import …/contrib/bunx
```

应用 bootstrap 构造 `Deps{DB: db}` 处不变（*bun.DB 装入 any 字段）。

### 4. go.mod

```sh
go mod edit -droprequire github.com/lynx-go/lynx-start/kernel \
            -require github.com/lynx-go/lynx-start@v0.5.0 \
            -require github.com/lynx-go/lynx-start/contrib/bunx@v0.1.0
go mod tidy
```

grpcapi 统一到 v0.4.0；bun 保持 v1.3.0。

### 5. 模块分发侧

安装 CLI 更名 **lynx-mods**（`go install github.com/lynx-go/lynx-mods/cli/cmd/lynx-mods@latest`），
registry 迁至 lynx-go/lynx-mods；应用内 `lynx-start.json`→`lynx-mods.json`、
`.lynx-start/`→`.lynx-mods/`。已装模块的应用按 lynx-mods 仓的升级指引
逐模块 `add <m> -o` 重建基线（金丝雀实例：lynx-start-demo 的迁移提交）。
