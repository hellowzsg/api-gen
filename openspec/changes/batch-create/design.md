## Context

api-gen 的实体级 Create 形态自 apigen-core 起固定为单条创建：Create 请求内嵌各资源（可选，支持部分创建）、响应回带 key。`create-client-key` 提案扩展了 `create.key` 配置（server/client 两种模式），但仍只生成单条 Create RPC。

首个重度使用方 my-wechat 项目的导入场景需要批量写入：解密后的数据按批次导入，逐条 Create 的网络往返成本不可接受。当前工具无批量创建能力，导入脚本只能循环单条 Create，性能瓶颈严重。

AIP-233 定义了批量创建的标准模式（`BatchCreate` RPC），api-gen 需要补齐这一能力。

## Goals / Non-Goals

**Goals:**
- `create` 增加 `batch` 布尔字段，声明是否额外生成 BatchCreate RPC
- BatchCreate Request 复用单条 Create 的 Request 形态作为 `requests` repeated 元素
- BatchCreate Response 返回 `repeated <KeyType> keys`
- HTTP 路由 `POST /{collection}/batchCreate body:"*"`（与 BatchGet/List 对称）
- `batch: true` 可与 `key: server` / `key: client` 任意组合
- 默认行为完全不变（`batch` 默认 false），向后兼容

**Non-Goals:**
- 不改变单条 Create 的任何生成形态
- 不定义批量创建的事务语义（全部成功 vs 部分成功）——属于服务端实现层
- 不定义重复 key 的服务端行为（实现层返回 ALREADY_EXISTS，不属于代码生成）
- 不在资源级生成 BatchCreate（BatchCreate 是实体级，与 Create 同级）
- 不支持 `batch` 的 service 级 narrowing 独立控制（随 Create 一起暴露/收窄）

## Decisions

### D1: 配置形态 `create: { batch: true }`，默认 false

**决策**：`CreateDef` 新增 `Batch bool` 字段，`create: { batch: true }` 时额外生成 `BatchCreate<Entity>s` RPC。

**理由**：
- `batch` 放在 `create` 内而非独立顶层配置：BatchCreate 是 Create 的批量变体，语义同源
- 布尔字段无需取值校验（YAML 解析器天然保证只有 true/false）
- 默认 false 保证存量配置零影响

### D2: BatchCreate Request 复用 CreateRequest 形态

**决策**：`BatchCreate<Entity>sRequest { repeated Create<Entity>Request requests = 1 }`；Response `{ repeated <KeyType> keys = 1 }`。

**理由**：
- 复用 `Create<Entity>Request` 作为 `requests` 元素，避免重复定义相同字段布局
- client-key 模式下每个 `Create<Entity>Request` 携带各自的 key，天然支持批量异构 key
- Response `repeated keys` 与单条 Create 的单 key 响应对称
- 部分创建语义不变：每个 `Create<Entity>Request` 内的各资源字段依然可选

### D3: HTTP 路由 `POST /{collection}/batchCreate body:"*"`

**决策**：HTTP 路由为 `POST /{prefix}/{Service}/{collection}/batchCreate body:"*"`，不携带 key 路径段。

**理由**：
- 与 BatchGet 的 `/{resource}/batchGet`、List 的 `/{resource}/list` 路径风格对称
- 集合级操作，key 在 body 的 `requests` 内，不在 path 携带
- 即使 client-key 模式也不在 path 携带 key（批量操作的 keys 在 body 内，path 无法表达多个 key）
- POST + `body:"*"` 与 BatchGet/List 一致，避免 query 序列化复杂性

### D4: Service 级 narrowing 随 Create

**决策**：BatchCreate 随 Create 一起暴露/收窄。service 级 `resources` narrowing 不影响 BatchCreate（BatchCreate 是实体级，不依赖资源级 narrowing）。

**理由**：
- BatchCreate 与 Create 同为实体级方法，narrowing 逻辑一致
- 不存在"暴露 BatchCreate 但不暴露 Create"的合理场景

### D5: api-linter 豁免 `core::0233` 系列

**决策**：BatchCreate 预期触发 `core::0233` 系列告警（与 BatchGet 触发 `core::0231` 对称），实施时在 internal/lint 豁免表中补充：
- `core::0233::request-message-name`（BatchCreate wrapper 命名）
- `core::0233::http-body`（POST + `body:"*"`）
- `core::0233::http-method`（POST）

**理由**：豁免应精确到实际触发的规则，与 BatchGet/List 的豁免风格一致。

## Risks / Trade-offs

### R1: 批量创建的事务语义未定义
- **风险**：批量创建中部分成功的行为（全部回滚 vs 部分成功）不在生成契约内
- **缓解**：定位为服务端实现层语义，文档注明；不在本提案展开

### R2: `batch` 字段一经发布不可移除
- **风险**：移除 `batch: true` 会导致 BatchCreate RPC 消失，属于 breaking change
- **缓解**：模式选择属于实体建模决策，文档中明确"一经发布不要移除 batch"

### R3: BatchCreate Request 元素复用 CreateRequest 的字段号兼容性
- **风险**：若 Create 的 Request 形态变化（如 server ↔ client 模式切换），BatchCreate 的 requests 元素形态同步变化
- **缓解**：Create 模式切换本身就是 breaking change（已由 create-client-key 提案约束），BatchCreate 天然跟随

### R4: 与活跃变更的潜在冲突
- **风险**：`create-client-key` 和 `custom-streaming` 均已实施完毕，本提案触及 `examples/book/api.yaml` 和 `testcase/fixtures/book/api.yaml`
- **缓解**：本提案在已实施完毕的基础上增量，无实际冲突

## Implementation Notes

### D2 修正：RequestsField.Type 用非限定名
实施时发现 `buildBatchCreate` 中 `RequestsField.Type` 若使用 `cfg.ResolveTypeName(requestName)` 会将 `CreateBookRequest` 解析为 `demo.business.book.CreateBookRequest`（实体 package），但 `CreateBookRequest` 是工具生成的 wrapper message，定义在各 service proto 文件内（不同 service 的 proto package 不同）。跨 service 引用时（如 AdminService 引用 book entity 的 BatchCreate），限定名指向的 package 不含该 message，导致 protocompile 失败。

修正：`RequestsField.Type` 直接使用非限定名 `CreateBookRequest`，因为该 message 与 `BatchCreateBooksRequest` 在同一 proto 文件中定义，proto 自动在同文件内解析。
