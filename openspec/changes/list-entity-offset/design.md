## Context

apigen 当前将 List 方法归为资源级方法（声明在 `resource.reader.list`），使用 AIP 标准的 `page_size` + `page_token` 游标分页。经过实际使用和语义审查，发现两个问题：

1. **List 语义上属于 Entity 级**：List 是对实体集合的查询，不以具体 entity key 为前提。当前声明在 resource 下导致 HTTP 路径包含 resource 段（`/{entity}/{resource}/list`），但实际上 List 不需要先定位某个具体实体。
2. **游标分页不适合内部服务**：`page_token` 是 opaque token，客户端无法跳页，服务端需维护游标状态。对于内部微服务场景，`limit + offset` 更直观、实现更简单。

### 现状

- `list` 声明在 `resource.reader` 下（`internal/yaml/parser.go:124-129`）
- `buildList()` 在 `buildResource()` 中调用（`internal/ir/builder.go:667-668`）
- `ListIR` 使用 `PageSize`/`PageToken`/`NextPageToken`/`TotalSize`（`internal/ir/builder.go:139-151`）
- HTTP 路径含 resource 段（`internal/ir/builder.go:530-537`）
- Service 收窄通过 `ReaderNarrowIR.List` 控制（`internal/ir/builder.go:285-288`）
- `design-v2.md` 定义 List 为资源级方法，字段号 `1=page_size, 2=page_token, 3=filter, 4=order_by`

## Goals / Non-Goals

**Goals:**
1. 将 List 从 resource 级提升到 entity 级
2. 分页从 `page_size + page_token` 改为 `limit + offset`
3. 移除 `next_page_token`，`total_size` 改为必选
4. HTTP 路径对齐 entity 级模式（无 resource 段、无 key 段）
5. Service 收窄机制适配 entity 级 List

**Non-Goals:**
1. 不支持一个 entity 多个 resource 的 List（一个 entity 只有一个 list 声明）
2. 不保留双分页模式（直接替换，不提供 `pagination: cursor|offset` 选项）
3. 不改变 `filter_type` / `order_by` 的现有语义

## Decisions

### D1: List 声明位置——Entity 级 `list` 块

**决策**：在 `Entity` 下新增 `list` 块，包含 `resource`（必填）和 `list_config`（可选）。

```yaml
entities:
  - name: book
    list:
      resource: meta
      list_config:
        total_size: true
        filter_type: BookMetaFilter
```

**理由**：
- List 是集合级操作，不以具体 key 为前提，与 Create/BatchCreate/Delete 同级
- `list.resource` 指明 List 返回的是哪个 resource 的数据，保留与 resource 类型的关联
- 一个 entity 只允许一个 `list` 声明。如果未来需要 List 多个 resource，可通过后续提案扩展为 `list: [{ resource: meta, ... }, { resource: content, ... }]`，但当前不过度设计

**被否决的方案**：
- *方案 B：List 留在 resource 级但语义标注为 entity-level* — 不采用，YAML 声明位置应反映实际语义层级，否则增加认知负担
- *方案 C：Entity 级 `list: true` + resource 级 `list_config`* — 不采用，配置分散在两处，不直观

### D2: 分页字段——limit + offset，移除 next_page_token

**决策**：

Request:
```
int32 limit     = 1;  // 每页数量
int32 offset    = 2;  // 偏移量（从 0 开始）
<FilterType> filter = 3;
string order_by = 4;
```

Response:
```
repeated <Resource> <resource>s = 1;
int32 total_size = 2;
```

**理由**：
- `limit`/`offset` 直接映射 SQL `LIMIT N OFFSET M`，服务端实现零转换
- 移除 `next_page_token`：offset 分页下客户端可直接计算下一页 offset（`offset + limit`），无需服务端返回 token
- `total_size` 改为必选：offset 分页下客户端需要 total_size 计算总页数和判断是否还有更多数据。原 `list_config.total_size` 配置项移除（不再可选关闭）
- 字段号 3（filter）和 4（order_by）保持不变，减少 proto 兼容性影响

**被否决的方案**：
- *方案 B：双模式 `pagination: cursor|offset`* — 不采用，项目尚无外部消费者，不需要兼容旧分页模式，双模式增加复杂度
- *方案 C：保留 next_page_token 作为可选字段* — 不采用，offset 分页下该字段无意义，保留只会造成困惑

### D3: HTTP 路径——Entity 级，无 resource 段

**决策**：

```
POST /{prefix}/{svc}/{entity}/list  body:"*"
```

**理由**：
- List 是集合级操作，不绑定具体 entity key，路径中不应有 key 段
- 路径不含 resource 段：resource 信息通过 request body 的字段语义体现（返回的是 `<resource>s`），URL 中不需要重复
- 与 BatchCreate 路径模式对齐：`/{prefix}/{svc}/{entity}/batchCreate`

**对比当前路径**：
- Before: `POST /library/LibraryService/book/meta/list`
- After:  `POST /library/LibraryService/book/list`

### D4: Service 收窄——Entity 级 `list` 布尔

**决策**：`ServiceEntityIR` 新增 `List *bool` 字段，控制 List 是否在该 service 中暴露。

```yaml
services:
  - name: AdminService
    entities:
      - name: book
        list: true    # 显式开启
      - name: shelf
        list: false   # 显式关闭
```

**收窄规则**：
- `list: true` → 暴露 List（前提：entity 声明了 list）
- `list: false` → 不暴露 List
- 省略 → 继承 entity 声明（entity 有 list 则暴露）

**被否决的方案**：
- *方案 B：resource 级收窄不变（保留 `reader.list`）* — 不采用，List 已提升到 entity 级，收窄机制应跟随

### D5: `list_config.total_size` 移除

**决策**：`ListConfig.TotalSize` 字段移除，`total_size` 在 Response 中总是生成。

**理由**：offset 分页下 `total_size` 是必要信息（客户端计算分页），不应可选关闭。移除该配置项简化 schema。

### D6: ListIR 结构体变更

**Before:**
```go
type ListIR struct {
    RPCName        string
    RequestName    string
    ResponseName   string
    PageSize       FieldIR
    PageToken      FieldIR
    Filter         FieldIR
    OrderBy        FieldIR
    ResourcesField FieldIR
    NextPageToken  FieldIR
    TotalSize      *FieldIR  // 可选
    HTTPAnnotation *HTTPAnnotation
}
```

**After:**
```go
type ListIR struct {
    RPCName        string
    RequestName    string
    ResponseName   string
    Limit          FieldIR
    Offset         FieldIR
    Filter         FieldIR
    OrderBy        FieldIR
    ResourcesField FieldIR
    TotalSize      FieldIR  // 必选
    HTTPAnnotation *HTTPAnnotation
}
```

### D7: 生成 RPC 命名不变

RPC 名仍为 `List<Entity><Resource>s`（如 `ListBookMetas`），因为：
- `list.resource` 指明了目标 resource，RPC 名中的 resource 部分来源于此
- 命名不变减少客户端认知变化，仅字段和路径变化

### D8: api-linter 豁免

List 改为 entity 级后，HTTP 路径从 `/{entity}/{resource}/list` 变为 `/{entity}/list`。需检查 api-linter 规则 `core::0231`（HTTP 方法）和 `core::0132`（HTTP body）是否对新路径报错。

预期无需新增豁免：
- List 仍用 POST + `body:"*"`，符合 AIP-132 BatchGet/List 用 POST 的规范
- 路径不含 resource 段不影响 api-linter 校验（api-linter 不校验路径层级深度）

## Implementation Notes

### 偏差：reader.http 移除

实现过程中发现 `reader.http`（per-method HTTP override）原本仅用于 List 方法（根据源码注释 "reader.http override applies only to List"）。List 提升到 entity 级后，`reader.http` 无可应用的目标方法。因此实现中移除了 `ReaderDef.HTTP` 字段及其校验逻辑。

影响范围：
- `internal/yaml/parser.go` — `ReaderDef` 移除 `HTTP *HTTPOverride` 字段
- `internal/yaml/validate.go` — `validatePerMethodHTTPOverrides` 移除 reader.http 分支
- `internal/ir/builder.go` — `fillResourceAnnotations` 移除 List 的 reader.http override 逻辑
- 相关测试（`TestParseHTTPOverride`、`TestBuildPerMethodHTTPOverride`、`TestValidateHTTPOverridePath` 等）更新为仅测试 `writer.update.http`

`writer.update.http` 和 `custom_methods[].http` 保持不变。

### 偏差：D2 字段号确认

实现确认 Request 字段号：`limit=1, offset=2, filter=3, order_by=4`；Response 字段号：`<resource>s=1, total_size=2`。filter 和 order_by 字段号与旧方案（page_size=1, page_token=2, filter=3, order_by=4）保持一致，减少 proto 兼容性影响。

### 测试策略

由于改动是跨层破坏性变更（YAML schema → IR → render → examples → testcase），TDD 的 "先写失败测试" 步骤简化为 "修改测试使其匹配新 schema，验证编译通过后运行"。所有内部测试（yaml/ir/render/cli/build/dep）均通过，examples/book e2e 测试通过，testcase positive/negative 测试通过。

## Risks

### R1: 破坏性变更影响范围

**风险**：YAML schema、生成 proto、HTTP 路径全部变更，所有使用 List 的 example/testcase 需同步更新。

**缓解**：项目尚在开发阶段，无外部消费者。变更范围可控（当前仅 `examples/book` 和 `testcase/fixtures/book` 使用 List）。

### R2: Entity 多 resource 场景的 List 覆盖

**风险**：当前设计一个 entity 只能 List 一个 resource。如果 entity 有 `meta` 和 `content` 两个 resource，只能 List 其中一个。

**缓解**：实际业务中，List 通常只针对主 resource（如 `meta`），`content` 等大字段资源不需要列表查询。如果未来有需求，可通过后续提案扩展为 list 列表。

### R3: offset 分页大数据量性能

**风险**：`OFFSET 100000` 在大数据量下性能差（数据库需扫描跳过的行）。

**缓解**：这是服务端实现问题，不在代码生成工具的职责范围内。工具只负责生成接口定义，分页性能由服务端实现决定。如需游标分页优化，可后续提案添加。
