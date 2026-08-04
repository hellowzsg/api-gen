# List Books + EntityItem — 设计

## Context

上一轮 `list-entity-offset` 已将 List 提升为实体级能力。本轮解决两个遗留语义问题：
1. RPC 命名仍带 resource 段，与实体级语义不符
2. `list.resource` 仅支持单值，无法一次返回多数据面

用户明确决策：
- RPC 名改为 `List<Entity>s`（如 `ListBooks`）
- `list.resource` 改为列表 `[meta, content]`
- Response 用工具自动生成的 `<Entity>Item` message，含所有声明资源字段
- 走完整 OpenSpec 流程

## Goals

1. `list.resource` 支持多资源（`[]string`）
2. RPC 命名纯实体级：`ListBooks`
3. Response 返回 `<Entity>Item` 聚合类型

## Non-Goals

- 不生成 per-resource 的多个 List RPC
- 不改分页语义（仍 limit+offset，实体维度）
- 不改 HTTP 路径、Service 收窄、`list_config.filter_type`

---

## Decisions

### D1: `list.resource` 改为 `[]string`

`EntityListDef.Resources []string`（YAML tag 仍为 `resource`），支持 `resource: [meta, content]`。

- **校验**：列表至少一个元素；每个元素必须是实体已声明的资源名；元素不可重复；引用不存在资源 → fail-fast 报错。

### D2: RPC 命名为 `List<Entity>s`

- `RPCName` = `"List" + entityPascal + "s"`（`ListBook` → `ListBooks`）
- `RequestName` = `ListBooksRequest`，`ResponseName` = `ListBooksResponse`
- 不再从 resource 名派生方法名

### D3: 生成 `<Entity>Item` 聚合类型

- 工具自动生成 `<Entity>Item`（如 `BookItem`），`ItemName` 存于 `ListIR`
- `ItemFields []FieldIR`：每个声明的资源按声明序分配字段号 1..N
  - 字段名 = 资源名（`meta`、`content`），类型 = 资源 `type_` 的 ResolveTypeName
- `ListBooksResponse`：
  ```proto
  message ListBooksResponse {
    repeated BookItem items = 1;
    int32 total_size = 2;
  }
  ```

### D4: `ListIR` 结构调整

```go
type ListIR struct {
    RPCName        string
    RequestName    string
    ResponseName   string
    ItemName       string
    Limit          FieldIR
    Offset         FieldIR
    Filter         FieldIR
    OrderBy        FieldIR
    ItemFields     []FieldIR   // 替代原 ResourcesField
    TotalSize      FieldIR
    HTTPAnnotation *HTTPAnnotation
}
```

- 移除 `ResourcesField FieldIR`（不再存在单一资源 repeated 字段）

### D5: `buildList` 签名变更

`buildList(entityPascal string, resources []apigenyaml.Resource, cfg *apigenyaml.Config, lc *ListConfig) *ListIR`

- 遍历 resources 构建 `ItemFields`（字段号按传入顺序 1..N，类型用 `cfg.ResolveTypeName` 解析）
- 校验所有资源均解析到（调用方在 `buildEntity` 中按 `list.resource` 名收集，validate 已兜底），否则返回错误

### D6: HTTP 注解与路径不变

- `buildListAnnotation()` 不变：`POST /{prefix}/{Service}/{collection}/list`，`body:"*"`
- Service 收窄（`entities[].list: true/false`）逻辑不变

### D7: filter/order_by 不变

- `list_config.filter_type` 仍控制 `filter` 字段类型（默认 `string`）
- `order_by` 恒为 `string`，字段号 3/4

---

## Risks

### R1: Response 字段号破坏

Response 从 `{ repeated metas = 1, total_size = 2 }` 变为 `{ repeated BookItem items = 1, total_size = 2 }`，且需要额外生成 `BookItem` 类型。属预期 breaking change（项目无外部消费者）。

### R2: 多资源 Item 的字段冲突

若实体两个资源重名（不可能，资源名唯一），或资源 `type_` 解析失败，`buildList` 需 fail-fast。已通过 D1 校验 + D5 解析兜底。

### R3: 语义混淆

`ListBooks` 返回 `BookItem`（含 meta+content）意味着一次列表同时拉取多数据面。若未来只想要单一数据面，可考虑 filter 或独立 RPC，本期不处理。

---

## Implementation Notes

### 偏差：`buildList` 实现细节

实现确认 `buildList(entityPascal string, resources []apigenyaml.Resource, cfg *apigenyaml.Config, lc *ListConfig) *ListIR`。ItemFields 的字段名 = 资源名（如 `meta`/`content`），类型 = `cfg.ResolveTypeName(r.Type)`。

### 偏差：Service 收窄与 BatchGet

为让 testcase/positive 的 AdminService 收窄断言（无 BatchGet）成立，fixtures/book 的 AdminService 配置了 `resources: [{name: meta, reader: {}}]`，收窄掉 meta 的 batch 与 content 资源。这与 examples/book 的 AdminService 设计保持一致。

### 生成形态确认（examples/book）

```proto
rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);
message ListBooksRequest  { int32 limit=1; int32 offset=2; BookMetaFilter filter=3; string order_by=4; }
message ListBooksResponse { repeated BookItem items=1; int32 total_size=2; }
message BookItem         { BookMeta meta=1; BookContent content=2; }
```

HTTP 路径 `POST /{prefix}/{Service}/{collection}/list` 不变。Service 收窄（`entities[].list`）、`list_config.filter_type`、limit+offset 分页语义均不变。
