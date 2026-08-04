## MODIFIED Requirements

### Requirement: 实体级 List 方法生成

apigen 必须为声明了 `list` 块的实体生成实体级 List 方法。`list` 块声明在 entity 级别，通过 `list.resource`（列表）指定 List 操作的目标资源集合。

#### Scenario: List 方法生成（单资源）

- **WHEN** 实体声明 `list: { resource: [meta] }`
- **AND** 该实体下存在名为 `meta` 的 resource 声明
- **THEN** 生成 `List<Entity>s` RPC（如 `ListBooks`），Request 为 `ListBooksRequest`（`limit=1, offset=2, filter=3, order_by=4`），Response 为 `ListBooksResponse`（`repeated BookItem items=1, total_size=2`）

#### Scenario: List 方法生成（多资源）

- **WHEN** 实体声明 `list: { resource: [meta, content] }`
- **AND** 实体下存在 `meta` 与 `content` 两个 resource
- **THEN** 生成 `ListBooks` RPC，Response 为 `ListBooksResponse`（`repeated BookItem items=1, total_size=2`），并自动生成 `<Entity>Item` 类型（如 `BookItem`），字段为声明序的每个资源（`meta=1, content=2`）

#### Scenario: RPC 命名

- **WHEN** 实体 `book` 声明 `list`
- **THEN** RPC 名为 `ListBooks`（实体复数），Request/Response 分别为 `ListBooksRequest` / `ListBooksResponse`，不再包含 resource 段

#### Scenario: list.resource 列表校验

- **WHEN** `list.resource` 为空列表或引用不存在/重复的 resource
- **THEN** apigen fail-fast，输出包含实体与资源上下文的错误

## ADDED Requirements

### Requirement: EntityItem 聚合类型生成

apigen 必须为带 List 的实体自动生成 `<Entity>Item` 聚合 message 类型，用于在单个 List 响应中返回多个数据面。

#### Scenario: EntityItem 字段

- **WHEN** 实体 `book` 声明 `list.resource: [meta, content]`
- **THEN** 生成 `BookItem` message，字段按声明序为 `BookMeta meta=1`、`BookContent content=2`

#### Scenario: Response 引用 EntityItem

- **WHEN** 生成 `ListBooksResponse`
- **THEN** 字段为 `repeated BookItem items=1`、`int32 total_size=2`

## REMOVED Requirements

### Requirement: 单值 list.resource 与资源级 List 命名

- 移除 `list.resource` 为单个 `string` 的声明形式（现为 `[]string`）
- 移除 `List<Entity><Resource>s` 的 RPC 命名（如 `ListBookMetas`），改为 `List<Entity>s`（如 `ListBooks`）
- 移除 List Response 直接返回单一 `repeated <resource>s` 字段的形式（现为 `repeated <Entity>Item items`）
