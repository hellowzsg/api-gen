## MODIFIED Requirements

### Requirement: 资源级方法生成

apigen 必须为每个资源的 reader/writer 生成资源级方法：Get/BatchGet/Update。List 不再属于资源级方法（已提升至实体级，见"实体级 List 方法生成"要求）。

#### Scenario: Get 方法生成

- **WHEN** 资源声明 `reader: {}`（默认）
- **THEN** 生成 `Get<Entity><Resource>` RPC，Request 含 `key=1`，Response 含 `<resource>=1` 和 `version=2`（kind≠NONE 时）

#### Scenario: BatchGet 方法生成

- **WHEN** 资源声明 `reader.batch: true`
- **THEN** 生成 `BatchGet<Entity><Resource>s` RPC，Request 含 `repeated <Key> keys=1`，Response 含 `repeated <resource>s=1`

#### Scenario: Update 方法生成

- **WHEN** 资源声明 `writer.update: {}`
- **THEN** 生成 `Update<Entity><Resource>` RPC，Request 含 `<resource>=1, key=2, update_mask=3, version=4`（version 在 kind≠NONE 时），Response 为 `Empty` 或 `{version=1}`（STRONG 时）

## ADDED Requirements

### Requirement: 实体级 List 方法生成

apigen 必须为声明了 `list` 块的实体生成实体级 List 方法。`list` 块声明在 entity 级别（非 resource 级别），通过 `list.resource` 指定 List 操作的目标 resource。

#### Scenario: List 方法生成

- **WHEN** 实体声明 `list: { resource: meta }`
- **AND** 该实体下存在名为 `meta` 的 resource 声明
- **THEN** 生成 `List<Entity><Resource>s` RPC（如 `ListBookMetas`），Request 含 `limit=1, offset=2, filter=3, order_by=4`，Response 含 `<resource>s=1, total_size=2`

#### Scenario: list.resource 必填校验

- **WHEN** 实体声明 `list: { list_config: { ... } }` 但未指定 `list.resource`
- **THEN** apigen fail-fast，输出错误提示 `list.resource` 为必填字段

#### Scenario: list.resource 指向不存在的 resource

- **WHEN** 实体声明 `list: { resource: nonexistent }`，但实体下无名为 `nonexistent` 的 resource
- **THEN** apigen fail-fast，输出错误提示 `list.resource` 指向的 resource 不存在

#### Scenario: list_config.filter_type 自定义类型

- **WHEN** 实体声明 `list: { resource: meta, list_config: { filter_type: BookMetaFilter } }`
- **THEN** 生成的 List Request 中 `filter` 字段类型为 `BookMetaFilter`（而非默认 `string`），字段号仍为 3

#### Scenario: list_config.filter_type 省略默认 string

- **WHEN** 实体声明 `list: { resource: meta }` 但未指定 `filter_type`（或为空）
- **THEN** 生成的 List Request 中 `filter` 字段类型为 `string`

#### Scenario: list_config.filter_type 语法校验

- **WHEN** 实体声明 `list.list_config.filter_type` 为空字符串或以 `.` 或数字开头
- **THEN** apigen fail-fast，输出错误提示 filter_type 值非法

### Requirement: List 分页使用 limit + offset

List 方法的 Request 必须使用 `limit`（int32）+ `offset`（int32）分页，Response 必须返回 `total_size`（int32）。不再使用 `page_size`/`page_token`/`next_page_token`。

#### Scenario: Request 分页字段

- **WHEN** 实体声明了 `list`
- **THEN** 生成的 List Request 含 `int32 limit = 1;` 和 `int32 offset = 2;`，无 `page_size` 和 `page_token` 字段

#### Scenario: Response 分页字段

- **WHEN** 实体声明了 `list`
- **THEN** 生成的 List Response 含 `repeated <Resource> <resource>s = 1;` 和 `int32 total_size = 2;`，无 `next_page_token` 字段

#### Scenario: total_size 总是生成

- **WHEN** 实体声明了 `list`（无论 `list_config` 是否存在或是否含 `total_size`）
- **THEN** 生成的 List Response 总是包含 `total_size = 2` 字段（不可关闭）

### Requirement: List HTTP 路径为实体级

List 方法的 HTTP 路径必须为实体级路径（不含 resource 段、不含 key 段），与 BatchCreate 路径模式对齐。

#### Scenario: List 默认 HTTP 路径

- **WHEN** 实体声明了 `list`，且 HTTP 已启用，且无 `reader.http` 覆盖
- **THEN** 生成的 `google.api.http` 注解为 `post: "/<prefix>/<svc>/<entity>/list"` 且 `body: "*"`

#### Scenario: List HTTP 路径不含 resource 段

- **WHEN** 实体声明 `list: { resource: meta }`
- **THEN** HTTP 路径为 `/<prefix>/<svc>/<entity>/list`，不包含 `/meta/` 段

### Requirement: Service 级 List 收窄

Service 可通过 entity 级 `list` 布尔字段控制 List 方法是否在该 service 中暴露。

#### Scenario: list 省略时继承实体声明

- **WHEN** 实体声明了 `list`，且 service 的 entity 引用未指定 `list`
- **THEN** 该 service 暴露 List 方法（继承实体声明）

#### Scenario: list: true 显式开启

- **WHEN** 实体声明了 `list`，且 service 的 entity 引用指定 `list: true`
- **THEN** 该 service 暴露 List 方法

#### Scenario: list: false 显式关闭

- **WHEN** 实体声明了 `list`，且 service 的 entity 引用指定 `list: false`
- **THEN** 该 service 不暴露 List 方法

#### Scenario: 实体未声明 list 但 service 指定 list: true

- **WHEN** 实体未声明 `list`，但 service 的 entity 引用指定 `list: true`
- **THEN** apigen fail-fast，输出错误提示实体未声明 list 但 service 试图启用

## REMOVED Requirements

### Requirement: 资源级 reader.list 声明

~~资源可通过 `reader.list: true` 声明 List 方法，通过 `reader.list_config` 配置 List 参数。~~

此声明方式已移除。List 改为实体级声明（`entity.list`）。`reader.list` 和 `reader.list_config` 字段不再被解析。

### Requirement: List page_token 游标分页

~~List Request 使用 `page_size`（int32）+ `page_token`（string）游标分页，Response 返回 `next_page_token`（string）。`total_size` 可通过 `list_config.total_size: false` 关闭。~~

此分页模式已移除。改为 `limit` + `offset` 分页，`total_size` 必选生成。`list_config.total_size` 配置项不再存在。
