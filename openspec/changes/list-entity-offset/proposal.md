## Why

当前 List 方法存在两个设计问题：

### 问题 1：List 声明在 resource 级，语义不准确

List 是对实体集合的查询（"列出所有 book"），不以某个具体 entity key 为前提。但当前 `list: true` 声明在 `resource.reader` 下，与 Get/BatchGet/Update 并列——后三者都需要先指定 `{entity_id}` 才能操作某个 resource，而 List 不需要。

从 HTTP 路径也能看出语义错位：当前 List 路径为 `/{prefix}/{svc}/{entity}/{resource}/list`，但实际语义应是 `/{prefix}/{svc}/{entity}/list`（集合级操作，无 key 段）。对比 entity 级的 BatchCreate 路径 `/{prefix}/{svc}/{entity}/batchCreate`，List 更应与之对齐。

### 问题 2：分页使用 page_token 游标，不适合内部服务场景

当前 List 采用 AIP 标准的 `page_size` + `page_token` 游标分页，Response 回 `next_page_token`。这种模式适合面向外部公网的 API（opaque token，服务端可自由变更分页策略），但对于内部微服务场景：

- **offset 分页更直观**：客户端可直接跳页（`offset=20` 跳第 3 页），游标分页只能顺序翻页
- **total_size 已有**：既然 Response 已经返回 `total_size`，offset 分页与之天然配合（客户端可计算总页数）
- **实现更简单**：服务端直接 `LIMIT N OFFSET M`，无需维护游标状态
- **page_token 对内部服务无实际收益**：游标分页的核心优势（防止数据变更导致的重复/遗漏）在内部高一致性场景中价值有限

## What Changes

### 1. List 提升到 Entity 级

`list` 从 `resource.reader` 提升到 `entity` 级别，通过 `list.resource` 指定 List 操作的目标 resource。

**Before:**
```yaml
entities:
  - name: book
    key: { type_: BookId }
    resources:
      - name: meta
        reader:
          list: true
          list_config:
            total_size: true
            filter_type: BookMetaFilter
```

**After:**
```yaml
entities:
  - name: book
    key: { type_: BookId }
    list:
      resource: meta
      list_config:
        total_size: true
        filter_type: BookMetaFilter
    resources:
      - name: meta
        ...
```

- `list.resource` 必填，必须是该 entity 下已声明的 resource 名称
- 一个 entity 只能有一个 `list` 声明（List 针对一个目标 resource）
- `list_config` 语义不变（`total_size` 默认 true，`filter_type` 默认 string）

### 2. 分页改为 limit + offset

**Request 字段变更：**
```
Before:  page_size=1, page_token=2, filter=3, order_by=4
After:   limit=1,     offset=2,     filter=3, order_by=4
```

- `limit` (int32): 每页返回数量，语义同原 `page_size`
- `offset` (int32): 偏移量，从 0 开始
- `filter` 和 `order_by` 字段号不变（3、4）

**Response 字段变更：**
```
Before:  <resource>s=1, next_page_token=2, total_size=3(可选)
After:   <resource>s=1, total_size=2
```

- 移除 `next_page_token`
- `total_size` 字段号从 3 改为 2，且默认生成（不再可选关闭，offset 分页下 total_size 是必要信息）

### 3. HTTP 路径变更

List 从 resource 级路径变为 entity 级路径：

```
Before:  POST /{prefix}/{svc}/{entity}/{resource}/list  body:"*"
After:   POST /{prefix}/{svc}/{entity}/list             body:"*"
```

无 key leaves 段（List 是集合级操作，不绑定具体 entity key），与 BatchCreate 的路径模式对齐。

### 4. Service 收窄机制适配

`resource.reader.list` 收窄不再适用。新增 entity 级 List 收窄：

```yaml
services:
  - name: AdminService
    entities:
      - name: book
        list: true   # 或 false/省略
```

- `list: true` → 暴露 List（当 entity 声明了 list 时）
- `list: false` → 不暴露 List
- 省略 → 继承 entity 的 list 声明（有则暴露）

### 不在本次范围（Non-Goals）

- `filter_type` 的类型定制能力（已在 list-filter-type 提案实现，本次保持不变）
- `order_by` 的类型定制（保持 string）
- 双分页模式开关（直接替换为 limit+offset，不保留 page_token 选项）
- 一个 entity 多个 resource 的 List（当前只支持 List 一个 resource）

## Impact

### YAML Schema 变更

- `internal/yaml/parser.go`：
  - `Entity` 新增 `List *EntityListDef` 字段
  - 新增 `EntityListDef` 结构体（`Resource string` + `ListConfig *ListConfig`）
  - `ReaderDef` 移除 `List bool` 和 `ListConfig *ListConfig` 字段
  - `ServiceEntityIR` 收窄新增 `List *bool` 字段

### IR 变更

- `internal/ir/builder.go`：
  - `EntityIR` 新增 `List *ListIR` 字段
  - `ResourceIR` 移除 `List *ListIR` 字段
  - `ListIR` 结构体变更：`PageSize`/`PageToken` → `Limit`/`Offset`，移除 `NextPageToken`，`TotalSize` 从 `*FieldIR` 变为 `FieldIR`（必选）
  - `buildList()` 函数签名和逻辑调整
  - `buildEntity()` 中构建 List（从 `buildResource` 移出）
  - `fillResourceAnnotations` 中 List 相关代码移到 entity 级 HTTP 注解构建
  - 新增 `buildListAnnotation()` 构建 entity 级 List HTTP 注解

### 渲染变更

- `internal/render/template.go`：
  - `renderServiceRPCs` / `renderMessages`：List 渲染从 resource 循环移到 entity 级
  - List message 模板：`page_size` → `limit`，`page_token` → `offset`，移除 `next_page_token`，`total_size` 必选且字段号改 2
  - `narrowEntity`：List 收窄逻辑从 resource 级移到 entity 级

### 测试变更

- `testcase/fixtures/book/api.yaml`：List 声明从 resource 移到 entity 级
- `testcase/` 下所有涉及 List 的测试用例和 golden 文件更新
- `internal/ir/builder_test.go`：List 相关断言更新

### 示例变更

- `examples/book/api.yaml`：List 声明从 resource 移到 entity 级
- `examples/book/generated/`：重新生成所有产物

### 向后兼容性

**不向后兼容。** 这是一次破坏性变更：
- YAML schema 变更：`resource.reader.list` 不再被识别，必须改用 `entity.list`
- 生成的 proto 变更：字段名和字段号变化（page_size→limit, page_token→offset），移除 next_page_token
- HTTP 路径变更：`/{entity}/{resource}/list` → `/{entity}/list`

由于项目尚在开发阶段（无外部消费者），破坏性变更是可接受的。
