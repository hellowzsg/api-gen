## ADDED Requirements

### Requirement: Create Key Mode Configuration

实体级 `create` 配置必须支持可选的 `key` 字段，声明主键生成方：`server`（服务端生成，默认）或 `client`（客户端指定）。`create: {}` 与 `create: { key: server }` 等价，保持现状行为。

#### Scenario: 省略 key 字段默认为 server
- **WHEN** 实体声明 `create: {}`
- **THEN** 生成契约与现状完全一致：Create 请求仅含各资源字段（1..N），响应只含 `key = 1`

#### Scenario: 显式声明 server 模式
- **WHEN** 实体声明 `create: { key: server }`
- **THEN** 生成契约与 `create: {}` 完全一致

#### Scenario: 非法取值报错
- **WHEN** 实体声明 `create: { key: clinet }`（非法值）
- **THEN** validate 阶段 fail-fast，错误信息包含实体名与合法取值（server/client）

### Requirement: Client-Key Create Request Shape

`create: { key: client }` 时，`Create<Entity>Request` 必须以 `key = 1`（实体 key 类型）为第一个字段，各资源字段按声明序顺延为 2..N+1（均可选，部分创建语义不变）；响应保持 `{ <KeyType> key = 1 }`。

#### Scenario: 单资源实体的 client 模式请求
- **WHEN** 实体 `message`（key 类型 `MessageId`，资源 `meta`）声明 `create: { key: client }`
- **THEN** 生成 `CreateMessageRequest { MessageId key = 1; MessageMeta meta = 2; }` 与 `CreateMessageResponse { MessageId key = 1; }`

#### Scenario: 多资源实体资源字段顺延
- **WHEN** 含 meta/content 两个资源的实体声明 `create: { key: client }`
- **THEN** Create 请求字段为 `key = 1, meta = 2, content = 3`

### Requirement: Client-Key Create HTTP Route

HTTP 启用且 `create: { key: client }` 时，Create 的 HTTP 注解必须为 `post: "/{prefix}/{Service}/{collection}/{key叶子段...}"` 且 `body: "*"`；key 叶子段按既有规则递归展开（嵌套 message 逐层展开、WKT `google.protobuf.*` 视为不透明叶子）。

#### Scenario: 复合 key 展开为路径段
- **WHEN** `message` 实体（`MessageId { AccountId account_id; string session_username; int32 local_id }`）声明 `create: { key: client }` 且 HTTP 启用
- **THEN** HTTP 注解为 `post: "/mwx/MessageService/message/{key.account_id.wxid}/{key.session_username}/{key.local_id}" body: "*"`

#### Scenario: server 模式路径不含 key
- **WHEN** 实体声明 `create: {}` 或 `create: { key: server }` 且 HTTP 启用
- **THEN** HTTP 注解保持 `post: "/{prefix}/{Service}/{collection}" body: "*"`，不含 key 路径段
