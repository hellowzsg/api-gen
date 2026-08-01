## ADDED Requirements

### Requirement: Create Batch Configuration

实体级 `create` 配置必须支持可选的 `batch` 布尔字段，声明是否额外生成 BatchCreate RPC。`create: {}` 与 `create: { batch: false }` 等价，保持现状行为（仅生成单条 Create）。`create: { batch: true }` 时额外生成 `BatchCreate<Entity>s` RPC。

#### Scenario: 省略 batch 字段不生成 BatchCreate
- **WHEN** 实体声明 `create: {}` 或 `create: { key: client }`（省略 batch）
- **THEN** 仅生成单条 `Create<Entity>` RPC，不生成 BatchCreate

#### Scenario: batch: true 额外生成 BatchCreate
- **WHEN** 实体声明 `create: { batch: true }`
- **THEN** 同时生成 `Create<Entity>` 和 `BatchCreate<Entity>s` 两个 RPC

#### Scenario: batch 与 key 组合
- **WHEN** 实体声明 `create: { key: client, batch: true }`
- **THEN** 生成 client-key 模式的 `Create<Entity>` + `BatchCreate<Entity>s`，BatchCreate 的 requests 元素为 client-key 模式的 `Create<Entity>Request`（含 key=1 + resources=2..N+1）

### Requirement: BatchCreate Request Shape

`create: { batch: true }` 时，`BatchCreate<Entity>sRequest` 必须为 `{ repeated Create<Entity>Request requests = 1 }`；`BatchCreate<Entity>sResponse` 必须为 `{ repeated <KeyType> keys = 1 }`。

#### Scenario: server-key 模式的 BatchCreate 请求
- **WHEN** 实体 `book`（key 类型 `BookId`，资源 `meta`/`content`）声明 `create: { batch: true }`（server-key 模式）
- **THEN** 生成 `BatchCreateBooksRequest { repeated CreateBookRequest requests = 1; }` 与 `BatchCreateBooksResponse { repeated BookId keys = 1; }`，其中 `CreateBookRequest` 形态为 `{ BookMeta meta = 1; BookContent content = 2; }`

#### Scenario: client-key 模式的 BatchCreate 请求
- **WHEN** 实体 `note`（key 类型 `NoteId`，资源 `meta`）声明 `create: { key: client, batch: true }`
- **THEN** 生成 `BatchCreateNotesRequest { repeated CreateNoteRequest requests = 1; }`，其中 `CreateNoteRequest` 形态为 `{ NoteId key = 1; NoteMeta meta = 2; }`

### Requirement: BatchCreate HTTP Route

HTTP 启用且 `create: { batch: true }` 时，BatchCreate 的 HTTP 注解必须为 `post: "/{prefix}/{Service}/{collection}/batchCreate"` 且 `body: "*"`；不携带 key 路径段（即使是 client-key 模式）。

#### Scenario: server-key 模式 BatchCreate HTTP 路由
- **WHEN** 实体 `book` 声明 `create: { batch: true }` 且 HTTP 启用（prefix `/library`，service `LibraryService`）
- **THEN** HTTP 注解为 `post: "/library/LibraryService/book/batchCreate" body: "*"`

#### Scenario: client-key 模式 BatchCreate HTTP 路由不含 key
- **WHEN** 实体 `note` 声明 `create: { key: client, batch: true }` 且 HTTP 启用
- **THEN** HTTP 注解为 `post: "/library/LibraryService/note/batchCreate" body: "*"`，不含 key 路径段

### Requirement: BatchCreate Service Narrowing

BatchCreate 随 Create 一起暴露/收窄。当 service 级 narrowing 导致 Create 不生成时，BatchCreate 也不生成。service 级 `resources` narrowing 不影响 BatchCreate（BatchCreate 是实体级方法）。

#### Scenario: Create 暴露时 BatchCreate 一并暴露
- **WHEN** 实体声明 `create: { batch: true }` 且 service 暴露该实体（无 narrowing 限制 Create）
- **THEN** service 同时暴露 `Create<Entity>` 和 `BatchCreate<Entity>s`

#### Scenario: resources narrowing 不影响 BatchCreate
- **WHEN** 实体声明 `create: { batch: true }` 且 service 通过 `resources` narrowing 仅暴露部分资源
- **THEN** BatchCreate 仍然暴露（BatchCreate 是实体级，不受资源级 narrowing 影响）
