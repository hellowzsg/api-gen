# List Books + EntityItem — 提案

## Why（为什么）

上一轮 `list-entity-offset` 已把 List 提升为实体级能力（`entity.list`，limit+offset 分页，HTTP 路径 `/{collection}/list`），但仍有两点未达实体级语义的"纯粹"：

1. **RPC 命名仍带 resource 段**：List 虽然声明在实体级，但生成的方法名仍为 `List<Entity><Resource>s`（如 `ListBookMetas`），resource 名来自 `list.resource` 单值。这与"List 是实体级能力"的语义不一致——它应当以实体为粒度命名，如 `ListBooks`。

2. **`list.resource` 仅支持单个资源**：当前 `list.resource` 是 `string`，只能指定一个数据面。但列表查询往往需要一次性返回实体的多个数据面（如书的 `meta` + `content`）。应支持 `resource: [meta, content]` 列表。

## What Changes（改什么）

1. **`list.resource` 从 `string` 改为 `[]string`**：支持声明一个或多个目标资源，如 `resource: [meta, content]`。

2. **RPC 命名改为纯实体级**：`List<Entity><Resource>s` → `List<Entity>s`（如 `ListBookMetas` → `ListBooks`）。Request/Response 同名相应变化（`ListBooksRequest`/`ListBooksResponse`）。

3. **Response 返回 EntityItem 聚合**：工具自动生成 `<Entity>Item` 类型（如 `BookItem`），字段 = 每个声明的资源按声明序（meta=1, content=2）。`ListBooksResponse` 为 `{ repeated BookItem items = 1, int32 total_size = 2 }`。

### 生成形态示例

```yaml
entities:
  - name: book
    list:
      resource: [meta, content]
```

```proto
service LibraryService {
  rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);
}

message ListBooksRequest {
  int32 limit = 1;
  int32 offset = 2;
  BookMetaFilter filter = 3;   // list_config.filter_type 指定时
  string order_by = 4;
}
message ListBooksResponse {
  repeated BookItem items = 1;
  int32 total_size = 2;
}
message BookItem {
  BookMeta meta = 1;
  BookContent content = 2;
}
```

## Impact（影响）

**破坏性变更**（延续 `list-entity-offset` 的破坏范围）：
- YAML schema：`list.resource` 从字符串改为列表
- 生成 proto：RPC 名、Response 形态（`items` + `BookItem`）、Response 字段号变化
- HTTP 路径不变：仍为 `/{prefix}/{Service}/{collection}/list`（`POST` + `body:"*"`）
- Service 收窄不变：`entities[].list: true/false`

项目尚在开发阶段，无外部消费者，可接受。

## 不做的范围

- 不引入 per-resource 的独立 List RPC（如 `ListBookMetas`/`ListBookContents` 并存）
- 不做多资源合并的分页语义变化（`limit`/`offset`/`total_size` 仍针对实体维度）
- 不改动 `list_config.filter_type` 语义
