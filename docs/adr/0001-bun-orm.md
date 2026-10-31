# ORM 选用 bun

数据访问层采用 uptrace/bun（pgdialect + pgdriver）。决定性理由是资产继承：参考实现 torchwood 与 lynx-clean-template 均基于 bun，其 bunrepo/mapper 分层、基于 `bun.IDB` 的 unit-of-work 模式、UPDATE 显式列白名单护栏（torchwood 2026-09-08 零值列事故的产物）都是踩坑后的现成答案。GORM 链式语义魔法多、AutoMigrate 与 SQL 迁移哲学冲突；ent 的 codegen 心智负担高；sqlc 在动态列表过滤（AIP-160 风格）上是硬伤。

## Considered Options

- **bun（选定）**：与两个参考项目一致，`IDB` 接口最适合事务管理器模式，raw SQL 一等公民。
- **GORM**：社区与语料最大，但与参考栈零交集，零值更新坑更隐蔽。
- **ent**：privacy 层对多租户有独特价值，可作 fork 进阶方向，不作模板默认。
- **sqlc**：可直接读 goose 迁移做编译期校验，适合未来复杂报表的混合补充，不做主 ORM。

## 修订（2026-10-31，ADR-0009 D5）

bun 的地位从「内核绑定」降为「首选适配」：项目可能因各种原因不选 bun 作
ORM，lynx-start 核心不再绑定——`kernel.Deps.DB` 改为 ORM 中立的 `any`
句柄，bun 全家（连接引导/事务实现/泛型列表/PG 测试基建）下沉
`contrib/bunx` 子模块（独立 tag），其他 ORM 自带 contrib 式适配器即可
接入同一套契约（uow.Runner/crud 分页与排序基元均为中立件）。本 ADR 的
选型结论不变：bun 仍是 lynx 生态应用的默认与推荐 ORM。
