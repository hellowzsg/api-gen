## ADDED Requirements

### Requirement: HTTP 总开关与 URI 前缀

apigen 必须支持通过 `settings.http.enable` 控制 HTTP 叠加层启停，缺省 `false`（向后兼容 P0 纯 gRPC）；`settings.http.prefix` 提供可选 URI 前缀。

#### Scenario: HTTP 启用
- **WHEN** `api.yaml` 声明 `settings.http.enable: true`
- **THEN** apigen 生成 `google.api.http` 注解、追加 `google/api/annotations.proto` import、编译时调用 `protoc-gen-grpc-gateway` 生成 `*.pb.gw.go`

#### Scenario: HTTP 关闭（缺省）
- **WHEN** `api.yaml` 未声明 `settings.http` 或 `settings.http.enable: false`
- **THEN** apigen 行为与 P0 完全一致（无 HTTP 注解、无 annotations import、无 grpc-gateway 编译）

#### Scenario: URI 前缀
- **WHEN** `api.yaml` 声明 `settings.http.prefix: /library`
- **THEN** 所有 HTTP 路径以 `/library` 开头（如 `/library/LibraryService/book/{key.id}/meta`）

#### Scenario: URI 前缀缺省
- **WHEN** `api.yaml` 未声明 `settings.http.prefix`
- **THEN** HTTP 路径无前缀（如 `/LibraryService/book/{key.id}/meta`），不自动补斜杠

### Requirement: P1 body_style 限制

P1 仅支持 `body_style: wrapper`（`body:"*"`）；声明 `body_style: resource` 或 `generate_openapi: true` 时 fail-fast 提示 P2 支持。

#### Scenario: body_style wrapper（默认）
- **WHEN** `api.yaml` 声明 `settings.http.body_style: wrapper` 或未声明（缺省 wrapper）
- **THEN** 所有带 body 的 RPC 使用 `body:"*"`

#### Scenario: body_style resource 报错
- **WHEN** `api.yaml` 声明 `settings.http.body_style: resource`
- **THEN** apigen fail-fast，提示 `body_style: resource` 在 P2 支持

#### Scenario: generate_openapi 报错
- **WHEN** `api.yaml` 声明 `settings.http.generate_openapi: true`
- **THEN** apigen fail-fast，提示 OpenAPI 生成在 P2 支持

### Requirement: key 类型递归解析

HTTP 启用时，apigen 必须递归解析 key 类型树提取标量叶子字段，按点路径展开为 URL path 段。

#### Scenario: 简单 key（单标量字段）
- **WHEN** key 类型为 `message BookId { string id = 1; }`
- **THEN** 提取叶子 `id`，path 段为 `{key.id}`

#### Scenario: 复合 key（嵌套 message）
- **WHEN** key 类型含嵌套 `message BookId { Org org = 1; int32 id = 2; }` + `message Org { string oid = 1; int32 qid = 2; }`
- **THEN** 深度优先序提取叶子 `org.oid` → `org.qid` → `id`，path 段为 `{key.org.oid}/{key.org.qid}/{key.id}`

#### Scenario: WKT 视为不透明叶子
- **WHEN** key 类型含 `google.protobuf.Timestamp created_at = 1;`
- **THEN** `created_at` 整体视为单一叶子，path 段为 `{key.created_at}`，不递归进入 Timestamp 内部

#### Scenario: repeated 字段 fail-fast
- **WHEN** key 类型含 `repeated string tags = 1;`
- **THEN** apigen fail-fast，提示 repeated 字段不能参与 HTTP path 绑定

#### Scenario: map 字段 fail-fast
- **WHEN** key 类型含 `map<string, string> labels = 1;`
- **THEN** apigen fail-fast，提示 map 字段不能参与 HTTP path 绑定

#### Scenario: oneof 字段 fail-fast
- **WHEN** key 类型含 `oneof id { string sid = 1; int32 iid = 2; }`
- **THEN** apigen fail-fast，提示 oneof 字段不能参与 HTTP path 绑定

#### Scenario: 循环引用 fail-fast
- **WHEN** key 类型树存在循环引用（A → B → A）
- **THEN** apigen fail-fast，提示循环引用

#### Scenario: optional 标量视为普通叶子
- **WHEN** key 类型含 `optional string id = 1;`
- **THEN** `id` 视为普通叶子，path 段为 `{key.id}`

### Requirement: flat 风格 HTTP 路径映射

HTTP 启用时，apigen 必须按 flat 风格生成路径：`{prefix?}/{service}/{collection}/{key叶子段...}/{resource?}`。

#### Scenario: Create 路径
- **WHEN** 实体声明 `create: {}` 且 HTTP 启用
- **THEN** 生成 `POST /{prefix}/{service}/{collection}` + `body:"*"`

#### Scenario: Delete 路径
- **WHEN** 实体声明 `delete: {}` 且 HTTP 启用
- **THEN** 生成 `DELETE /{prefix}/{service}/{collection}/{key叶子段...}`，无 body

#### Scenario: DeleteSoft 路径
- **WHEN** 实体声明 `delete_soft: {}` 且 HTTP 启用
- **THEN** 生成 `POST /{prefix}/{service}/{collection}/deleteSoft` + `body:"*"`（key 走 body）

#### Scenario: Get 路径
- **WHEN** 资源声明 `reader: {}` 且 HTTP 启用
- **THEN** 生成 `GET /{prefix}/{service}/{collection}/{key叶子段...}/{resource}`，无 body

#### Scenario: BatchGet 路径
- **WHEN** 资源声明 `reader.batch: true` 且 HTTP 启用
- **THEN** 生成 `POST /{prefix}/{service}/{collection}/{resource}/batchGet` + `body:"*"`

#### Scenario: List 路径
- **WHEN** 资源声明 `reader.list: true` 且 HTTP 启用
- **THEN** 生成 `POST /{prefix}/{service}/{collection}/{resource}/list` + `body:"*"`

#### Scenario: Update 路径
- **WHEN** 资源声明 `writer.update: {}` 且 HTTP 启用
- **THEN** 生成 `PATCH /{prefix}/{service}/{collection}/{key叶子段...}/{resource}` + `body:"*"`

### Requirement: google.api.http 注解生成

HTTP 启用时，apigen 必须为每个 RPC 生成 `google.api.http` 注解。

#### Scenario: 注解格式
- **WHEN** HTTP 启用且方法有 HTTPAnnotation
- **THEN** RPC 定义内追加 `option (google.api.http) = { <verb>: "<path>" [body: "<body>"] };`

#### Scenario: annotations import
- **WHEN** HTTP 启用
- **THEN** 生成的 proto 文件 import `google/api/annotations.proto`

### Requirement: googleapis 依赖物化校验

HTTP 启用时，apigen 必须校验 `import_protos` 中已有 googleapis（任何来源），都没有则 fail-fast。

#### Scenario: googleapis 可达
- **WHEN** HTTP 启用且 `import_protos` 中有条目能提供 `google/api/annotations.proto`（path/git/bsr 任一）
- **THEN** 校验通过

#### Scenario: googleapis 缺失
- **WHEN** HTTP 启用且 `import_protos` 中无任何条目能提供 `google/api/annotations.proto`
- **THEN** apigen fail-fast，提示需在 `import_protos` 中声明 googleapis 依赖

### Requirement: grpc-gateway 编译

HTTP 启用时，`apigen build` 必须追加调用 `protoc-gen-grpc-gateway` 生成 `*.pb.gw.go`。

#### Scenario: gw.go 生成
- **WHEN** `apigen build` 且 HTTP 启用
- **THEN** subprocess 调用 `protoc-gen-grpc-gateway`（参数 `paths=source_relative`），在 `generated/go/<service>/` 下生成 `*.pb.gw.go`

#### Scenario: 插件未安装自动安装
- **WHEN** `protoc-gen-grpc-gateway` 未在 PATH 中
- **THEN** apigen 自动 `go install` 到 GOPATH/bin（版本由 apigen `go.mod` 锁定）

#### Scenario: HTTP 关闭时不调用 gateway
- **WHEN** `apigen build` 且 HTTP 关闭
- **THEN** 不调用 `protoc-gen-grpc-gateway`，无 `*.pb.gw.go` 产物

### Requirement: api-linter 豁免扩展（HTTP）

HTTP 启用时，apigen 必须按实际触发追加 HTTP 相关 api-linter 豁免。

#### Scenario: Create 触发 http-body 豁免
- **WHEN** HTTP 启用且实体声明 `create: {}`
- **THEN** 追加 `core::0133::http-body` 豁免

#### Scenario: BatchGet 触发 http 豁免
- **WHEN** HTTP 启用且有 BatchGet 方法
- **THEN** 追加 `core::0231::http-body` + `core::0231::http-method` 豁免

#### Scenario: List 触发 http 豁免
- **WHEN** HTTP 启用且有 List 方法
- **THEN** 追加 `core::0132::http-method` + `core::0132::http-body` 豁免

#### Scenario: DeleteSoft 触发 http 豁免
- **WHEN** HTTP 启用且有 DeleteSoft 方法
- **THEN** 追加 `core::0135::http-method` + `core::0135::http-body` 豁免

#### Scenario: HTTP 关闭时不追加 http 豁免
- **WHEN** HTTP 关闭
- **THEN** 不生成任何 `http-method`/`http-body` 豁免（P0 行为不变）

### Requirement: HTTP path 变量校验（防线 2）

HTTP 启用时，apigen 必须在 generate 阶段（防线 2 闭包 dry-run）校验 HTTP path 变量。

#### Scenario: 标量叶子真实存在性
- **WHEN** HTTP 启用且 key 递归解析提取叶子
- **THEN** 从 protocompile link 结果校验叶子字段真实存在（非"信任声明"）

#### Scenario: 校验失败 fail-fast
- **WHEN** key 递归解析遇到 repeated/map/oneof 或循环引用
- **THEN** apigen fail-fast，在 generate 阶段报错（不等到 build）

### Requirement: 确定性输出（HTTP 扩展）

HTTP 注解的生成必须确定性，保证多次运行 bit-identical。

#### Scenario: HTTP 注解确定性
- **WHEN** 同一 `api.yaml`（HTTP 启用）多次运行 `apigen generate`
- **THEN** 生成的 proto 文件中 HTTP 注解（谓词、路径、body）bit-identical
