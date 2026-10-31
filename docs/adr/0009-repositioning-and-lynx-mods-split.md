# ADR-0009: lynx-start 定位调整——web 开发基础库，模块分发迁往 lynx-mods

日期：2026-10-31
状态：已接受

## 背景与决策

lynx-start 自 ADR-0003 定稿以来是「Kernel 契约库 + 源码模块 registry +
安装 CLI」三位一体的 shadcn 式模块基建。随着 torchwood / fleetly 等
自有项目的成型，生态分工变得清晰，单一仓库承担了两种不同生命周期的
资产：**范式基座**（所有项目每天都 import 的契约与电池）与**分发业务**
（registry 内容、安装器、脚手架——只在建项目时使用）。两者拆分：

- **lynx-start = lynx 框架的 web 开发基础库**：规范自有项目统一开发范式，
  同时作为第三方可用的开源基建（补位 lynx 刻意不做的 web 层：鉴权原语/
  ORM 范式/错误码/配置约定/分页）。
- **lynx-go/lynx-mods = 类 shadcn/ui 的模块分发**：registry + 模块 +
  安装 CLI（含 init/generate/scaffold）。

生态四仓与依赖方向：`lynx-mods → lynx-start(核心+contrib) → {lynx, grpcapi}`。
dogfood 序列：lynx-start-demo（金丝雀）→ torchwood（PG+bun 路径）→
fleetly（SQLite+database/sql 路径）。

## 顶层教义（D9）：偏向即默认，开放即接缝

lynx-start 带我们的设计/使用偏向（proto-first API 面、bun 首选、goose
迁移、Argon2id、AIP 风格）作为零配置默认；同时**每个偏向必须有公共接缝**
（接口/端口/协议），第三方可在自己仓库实现替换或扩展，永不需 fork。
消费分层：`电池层（严格中立）⊂ 契约层（grpcapi 冻结词汇+宿主类型）⊂
偏向装配层`。分层由 `layers_guard_test.go` 机器固化。

## 决策账本

- **D1 CLI 整迁 lynx-mods**（含 init/generate/scaffold，二进制
  `lynx-mods`）：`init --preset` 与安装器在代码级交织（原 cli
  init.go:98-110 直调 registry.LoadPresets+cmdAdd），拆分需复制
  render/manifest/lock 内部件或跨仓 import `internal/`（Go 禁止）。
  范式权威不随模板走：脚手架渲染出的应用 import lynx-start、版本地板
  钉住契约——模板是载体，依赖关系才是权威（shadcn CLI 同样含 init）。
- **D2 根 module 化**：module path `github.com/lynx-go/lynx-start`
  （根 tag `v0.5.0`），根包名 `kernel` 不变——下游只改 import 行、零
  限定符替换，避免 sed 误伤 `kernelConstraint` 等标识符。goimports 可能
  猜包名为 `lynxstart` 需一次手动别名，是可接受的一类税。
- **D3 历史 tag 原样搬运 + CLI 双前缀兼容**：e2e 依赖
  `organization@v0.1.0` 载荷真实差异（无 organization_name），同内容重建
  或 filter-repo 均会破坏该场景；旧 tag 经 `git fetch` 原样引入 lynx-mods，
  安装器的前缀改写集合化（新 `lynx-mods/modules/` 主 + 旧
  `lynx-start/modules/` 兼容），同时覆盖 `.go` import 与 `.proto`
  go_package 两条改写路径及旧 tag item.json 的 modulePath 对账。
- **D4 版本策略**：8 模块 0.1.x→0.2.0、CLI→cli/v0.3.0、
  kernelConstraint→`>=0.5.0`、grpcapi 全线统一 v0.4.0（含模块 buf.yaml
  的 BSR dep）、bun 统一钉 v1.3.0（Limit/Offset int64 签名教训）；
  advisories 保留历史 ID `LYNXSTART-0001`（不可变标识），新公告用
  `LYNXMODS-` 前缀。
- **D5 ORM 无关化**：`Deps.DB` 由 `*bun.DB` 改 `any`（ORM 中立句柄，
  类型由适配子模块约定），`contrib/bunx` 提供 `DB()/MustDB()` 类型化
  出口；uow（Runner/Isolator 纯接口）与 crud（分页/排序基元）留在核心。
  **否决 `Module[DB any]` 泛型方案**：方法级类型参数限制 + 模块分发落地
  成本过高；若未来出现真实的多元 ORM 模块生态再评估（演进通道开放）。
- **D6 根包纯契约化 + 三层依赖政策**：根包只留契约类型与 ValidateName
  级规则函数；`ValidateModules/AssembleAll/PolicyFiles/SortModules/
  Descriptors` 下沉 `compose/`，`DefaultChain` 下沉 `chain/`。依赖政策
  三层：**宿主框架**（lynx——身份性依赖，契约可引）／**冻结词汇库**
  （grpcapi——仅允许嵌入其 v0.3.0 起冻结的 SPI 面：interceptor.Stage+
  Order 常量、authn.Options/Chain/Authenticator/Enricher、server.
  Registration/ExtraRoute）／**可选实现**（bun 等——只准出现在 contrib/）。
  **否决把 grpcapi/lynx 抽象成平行接口**：那是重新发明生态基座（换掉
  它们=换生态），且中立词汇表将影子化对方已冻结的 SPI，零现实消费者。
- **D7 迁移纯度**：本迁移不夹带新电池/新 API；模式提取一律走
  「提取泛化→入 lynx-start→dogfood 换 import 采纳→版本化发布」循环
  独立立项（见 docs/battery-roadmap.md）。
- **D8 grpcapi 关系**：保持 peer，不合并（torchwood/fleetly 不经
  lynx-start 消费 grpcapi、其宿主无关性与 v1.0 验收独立、BSR proto 供给
  链事实源在 grpcapi 仓）、不中立化（如上）。修复既有版本分叉：本迁移
  统一 v0.4.0；政策：lynx-start 只在 grpcapi 冻结面 additive 变更时抬
  依赖。已知接缝（roadmap）：kernel/jwt 签发 HS256 vs grpcapi authn/jwt
  默认白名单 RS*/ES*。
- **D10 执行修正**：lynx-start-demo（现存七模块应用）纳入迁移作金丝雀；
  分层守卫机器化；遗留路径改名（`~/.cache/lynx-start`、e2e 别名）。
  环境变量所有权：`LYNXSTART_TEST_PG_DSN` 属 contrib/bunx/testutil
  （不改名，lynx-mods CI 必须设此名）；`LYNXSTART_REGISTRY`/
  `LYNX_E2E_MODULES` 属分发侧 → `LYNXMODS_*`。

### 范式定典（D11-D14，随迁移成文、经 dogfood 循环落地）

- **D11 分页多模式框架**：crud 定位为多模式列表框架——bounded（有界
  全取+ClampLimit）/ offset（页跳转，标注 O(n) 与并发漂移警示）/
  keyset（after 复合游标，插入并发稳定）三模式并列，每个 list 方法声明
  开放模式。**自定义 order_by** 一等能力：字段白名单+强制唯一 tiebreak；
  keyset 游标=编码「排序键值向量+order 摘要+filter 摘要」（encode-not-
  sign，无密钥仪式，换序/换滤自失效）。**自定义 filter** 开放泛型：
  `Params[F]` 携带每方法自有 filter 类型；核心只提供最小 typed 谓词集
  （等值/IN/区间+组合）；字符串 DSL（torchwood pkg/query 形态）留应用层。
  现行 AIP-158 签名 token 机制（PageTokenData/HMAC/TTL）从核心范式除名，
  随模块换装 crud v2 退场；crud 是库级辅助件而非契约强制（fleetly 不用
  亦不违范式）。
- **D12 数据访问电池族**（共享 order/tiebreak 词汇，遵守 D6 分层）：
  `storeerr` 存储错误归一（3× 证据：模块 pgdriver 23505 值类型坑/
  torchwood MapDocumentDBError/fleetly IsUniqueViolation 族——核心=领域
  哨兵集+瞬时/永久分类，contrib/bunx=pgdriver 映射）；`occ` 乐观并发/
  状态机 CAS（2×：torchwood current_version、fleetly Transit+generation）；
  行规约（时间/审计/tombstone 列+时钟格式约定+contrib/bunx hooks）；
  「四件一拍」写组合（状态 CAS+tombstone+outbox+审计同事务，fleetly
  ADR-0005——随 outbox 循环定典入 uow 文档）。
- **D13 机制沉淀三桶纪律**：考古桶（dogfood 重复证据）／蓝图桶（通用
  web 基建缺口：ratelimit/testkit/webhook 出站/blob/cache/lock/观测
  约定）／投机桶（只记触发器，不预建：OAuth+TOTP/quota 计量/feature
  flags/通道端口/搜索向量永属应用层）。生态已覆盖者引用不重建（重试/
  断路器/生命周期/选主/跨进程 bus/传输安全/配置加载/DI/CLI——归属 lynx
  与 grpcapi）。**电池准入门槛**：测试+dogfood 采纳+文档三件套齐才毕业
  入核心；弃用走 grpcapi 式 Deprecated 垫片节奏。蓝图桶项也须有消费者
  排期才动工。
- **D14 API 形态规范**（docs/api-shapes.md）：wire 上 ID 一律 string
  opaque（禁裸 int64）；时间用 `google.protobuf.Timestamp`（UTC）；枚举
  必含 `*_UNSPECIFIED=0`、「显式零值」用 optional、protojson 零值省略
  行为成文；**错误体定典 gRPC ErrorInfo detail 为权威投影**（proto
  envelope 仅 SDK/CLI 便利形态）；list 响应统一 meta（has_more+after
  回显；total_size 可选、禁默认 count）；幂等动作用带计数/状态响应优于
  Empty；gateway 映射行为成文（DELETE+Empty→200）；更新面显式动词优先
  （FieldMask 记 revisit）；路由资源名与字段裸 ID 分工成文；消息命名把
  buf lint 的 except 反转为正向规范。机器校验两级：句法级（buf lint/
  protovalidate 配置）+语义级（grpcapi guard 的 opt-in preset，由
  lynx-start 定义规则集），与 errcode v2/errstatus 同循环落地。

### 治理与 DX（D15）

SECURITY.md+GitHub 私密漏洞报告；UPGRADING.md（v0.4→v0.5 首版，demo
迁移即实证）；godoc 纪律（根包 doc.go+快速上手，电池 Example 基线）；
发版纪律：根模块 `v*` 与 contrib 子模块 `contrib/*/v*` 独立 tag 独立
节奏，联动仅在冻结面变更时；Go 最低版本由各 go.mod 声明（当前 1.26.8）；
覆盖率先记基线、门槛待 Phase 4 首循环后设定；examples/ 活文档随 testkit
循环起步。明确不做：兼容垫片（消费者仅 demo 且同批迁移）、CoC（暂缓）、
`x/` 实验命名空间（暂缓）。

## 后果

- 正面：核心零 ORM 绑定（机器验证：根 go.mod 与依赖树零 uptrace）；
  范式权威与分发业务解耦，各自独立演化与发版；开放性有接缝清单与机器
  守卫；两 dogfood 项目分别验证 bun 路径与 database/sql 路径。
- 负面/代价：module path 变更是一次性破坏（0.x 阶段 minor 破坏承诺内，
  tag `v0.5.0`）；`kernel` 包名与 `lynx-start` module 尾段不一致（工具
  链一次性别名税）；API 面替换 grpcapi 冻结 SPI（装配面中立化 Faces）
  记 revisit——出现真实非 proto-first 模块需求或 ConnectRPC 等新范式时
  再议。

## 关联

- ADR-0001（修订：bun 降为首选适配）、ADR-0003（修订：三层供给拆两仓）、
  ADR-0005（手写装配，不受影响）
- docs/battery-roadmap.md（三桶全景与准入纪律）、docs/extending.md
  （接缝清单与扩展指南）、docs/api-shapes.md（API 形态规范）
- lynx-go/lynx-mods：ADR-0006（承接决策与历史 tag 搬运）
