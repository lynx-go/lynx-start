# API 形态规范（Wire Shapes）——骨架

> 状态：既有实践的成文骨架（ADR-0009 D14）。完整规范与 grpcapi guard
> opt-in preset（机器校验）随 errcode v2/errstatus 循环落地；本文件先
> 固化已定之典，标注「待定」处留给该循环定稿。

本规范约束 proto-first API 的**wire 形态**（字段与消息的表达约定），
适用于 lynx 生态全部模块与应用。传输层协议（policy/网关/拦截）归
grpcapi，不在本文范围。

## 已定之典

- **ID**：wire 上一律 `string`、opaque——禁裸 int64 上 wire（JS 精度坑）；
  可读性（如 `web-42`）是服务端选择，客户端禁解析其语义。
- **时间**：`google.protobuf.Timestamp`（UTC）；存储格式属行规约
  （battery-roadmap ④），不属 wire 形态。
- **枚举**：必含 `*_UNSPECIFIED = 0`；「显式 false/零值」语义用
  `optional`；protojson 零值省略是特性不是 bug（消费端按缺省处理）。
- **错误体**：**gRPC ErrorInfo detail 是权威投影**（domain/reason +
  retryable/error_id metadata + BadRequest 字段违规）；proto envelope
  形态仅 SDK/CLI 侧便利投影。全文随 errcode v2/errstatus 定稿。
- **list 响应**：统一 meta——`has_more` + after 游标回显；
  `total_size` 可选、**禁默认 count**（代价警示）；形态与 crud v2
  三模式（D11）联动定稿。
- **幂等动作响应**：带计数/状态的结构优于 Empty（如 MarkAllRead 返回
  条数）；gateway 映射行为成文：DELETE+Empty → HTTP 200（非 204）。
- **更新面**：显式动词优先（ChangePassword/SetXxx）；FieldMask 记
  revisit（首个真批量更新需求触发）。
- **资源引用双形态**：路由用资源名（`/v1/organizations/{id}/tasks`，
  ADR-0008）；字段用裸 ID（`organization_id`）。分工已定，勿混用。
- **消息命名**：本生态 buf lint 对 RPC_RESPONSE_STANDARD_NAME 等取
  except（资源直出风格）——例外即事实规范，何时 `XxxResponse` 何时
  资源直出的正向化清单【待定，随 guard preset 出】。

## 机器校验（两级，随 errstatus 循环落地）

1. **句法级**：buf lint / protovalidate 配置（enum UNSPECIFIED、
   Timestamp 类型、optional 语义位）。
2. **语义级**：grpcapi guard 的 opt-in preset（本仓定义规则集）——
   list meta 形态、错误体投影、ID string、命名风格。preset 对
   lynx-mods 8 模块全量跑绿为验收。
