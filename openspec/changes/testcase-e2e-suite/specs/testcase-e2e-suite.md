## ADDED Requirements

### Requirement: testcase 目录结构

项目根目录下新建 `testcase/` 目录作为独立 Go module，包含 fixtures、positive、negative 三个子目录，系统化组织所有 e2e 测试。

#### Scenario: testcase 目录创建
- **WHEN** 开发者查看项目根目录
- **THEN** 存在 `testcase/` 目录，包含 `go.mod`、`fixtures/`、`positive/`、`negative/`、`README.md`

#### Scenario: testcase 独立 module
- **WHEN** 在 `testcase/` 目录下运行 `go test ./...`
- **THEN** 正向和反向测试均可独立运行，不依赖主 module 的 internal 包

### Requirement: 正向 e2e 测试套件

`testcase/positive/` 包含正向端到端测试，验证 `apigen generate/build` 生成的产物（proto、gRPC stub、HTTP gateway、OpenAPI spec）正确工作。

#### Scenario: generate 产物验证
- **WHEN** 对 book fixture 运行 `apigen generate`
- **THEN** 生成的 proto 文件包含正确的 service、RPC 方法、message 字段定义
- **AND** AdminService 收窄后只含指定方法，不含 BatchGet/GetContent/UpdateContent

#### Scenario: gRPC 全方法正向验证
- **WHEN** 使用 in-memory bufconn 启动 gRPC server 并调用 LibraryService 全部 9 个 RPC
- **THEN** 请求字段完整到达 server mock，响应字段完整返回 client
- **AND** version（uint64）、FieldMask、Filter、分页字段均正确序列化/反序列化

#### Scenario: HTTP gateway 全路由正向验证
- **WHEN** 使用 httptest 启动 grpc-gateway mux 并调用所有 HTTP 端点
- **THEN** Create=POST body=*、Delete=DELETE /{key.id}、Get=GET /{key.id}/resource 等 CRUD 路由均正确匹配
- **AND** custom method ArchiveBook 走 AIP-136 冒号语法路径
- **AND** protojson 序列化（uint64→string、FieldMask→逗号分隔）正确

#### Scenario: OpenAPI spec 正向验证
- **WHEN** 读取 `apigen build` 生成的 swagger.json
- **THEN** JSON 可被解析，paths 包含所有路由，HTTP verb 正确

#### Scenario: simple fixture P0 纯 gRPC 验证
- **WHEN** 对 simple fixture（无 HTTP 配置）运行 `apigen generate`
- **THEN** 生成的 proto 不含 `google.api.http` 注解
- **AND** 生成的 proto 不含 `import "google/api/annotations.proto"`
- **AND** 仅 gRPC service 定义，无 HTTP gateway 产物

#### Scenario: edge fixture 边界条件验证
- **WHEN** 对 edge fixture（多 entity、嵌套 key、WEAK version）运行 `apigen generate`
- **THEN** 多 entity 的各自 RPC 均正确生成
- **AND** 嵌套 key 类型（如 org.oid + id）的路径段正确绑定
- **AND** version.kind=WEAK 时生成 google.protobuf.UInt64Value wrapper 字段
- **AND** version.kind=NONE 时不生成 version 字段

### Requirement: 反向 e2e 测试套件

`testcase/negative/` 包含反向端到端测试，覆盖所有错误输入和异常路径。

#### Scenario: YAML 解析层错误
- **WHEN** 对缺少 syntax/name/entities 字段的 api.yaml 运行 generate
- **THEN** 返回对应的 "missing required field" 错误
- **AND** 包含未知 YAML 字段时返回 "not found" 错误

#### Scenario: 类型引用校验错误
- **WHEN** key.type_ 或 resource.type_ 为空、以点/数字开头、含非法字符
- **THEN** 返回 "type_ is empty" 或 "must start with letter" 或 "illegal character" 错误

#### Scenario: Service 引用校验错误
- **WHEN** service 引用不存在的 entity，或重复 entity name
- **THEN** 返回 "references nonexistent entity" 或 "duplicate entity name" 错误

#### Scenario: HTTP 配置校验错误
- **WHEN** HTTP 启用但无 googleapis 依赖，或 body_style 值非法
- **THEN** 返回 "no googleapis dependency found" 或 "invalid body_style" 错误

#### Scenario: 路径变量语法校验错误
- **WHEN** HTTP 路径含空变量 `{}` 或前导/尾随点 `{.id}` `{key.}`
- **THEN** 返回 "empty path variable" 或 "malformed path variable" 错误

#### Scenario: 插件校验错误
- **WHEN** plugins.js 值非 "es"
- **THEN** 返回 "unknown JS plugin" 错误

#### Scenario: IR 构建层错误
- **WHEN** HTTP 启用但 key descriptor 未提供，或 body_style: resource + 多 resource Create
- **THEN** 返回 "key descriptor not provided" 或 "ambiguous for Create" 错误

#### Scenario: key leaves 提取错误
- **WHEN** key 类型含 repeated/map/oneof 字段，或循环引用
- **THEN** 返回对应的 "cannot participate in HTTP path binding" 或 "circular reference" 错误

#### Scenario: 路径变量可达性错误
- **WHEN** HTTP 覆盖路径的 {key.xxx} 引用不存在的叶子
- **THEN** 返回 "not a reachable scalar leaf" 错误

#### Scenario: Option 注入校验错误
- **WHEN** option target 非法、target=field/message/rpc 但 path 为空、option name 为空或含非法字符
- **THEN** 返回对应的 "invalid option target" 或 "requires non-empty path" 或 "option name is empty" 错误

#### Scenario: CLI 生成层错误
- **WHEN** key.type_ 或 resource.type_ 在 proto 文件中找不到 descriptor
- **THEN** 返回 "descriptor not found in resolved protos" 错误

#### Scenario: HTTP 运行时反向路径
- **WHEN** 请求未注册的 HTTP 路由
- **THEN** 返回 404
- **WHEN** 对 POST-only 路由使用 PUT 方法
- **THEN** 返回 405 或 404
- **WHEN** 发送无效 JSON body
- **THEN** 返回 400
- **WHEN** 请求 AdminService 收窄后不存在的路由
- **THEN** 返回 404

#### Scenario: gRPC 运行时反向路径
- **WHEN** 在 AdminService 上调用未注册的方法
- **THEN** 返回 codes.Unimplemented
- **WHEN** 传入 nil 请求
- **THEN** 返回错误而非 panic
- **WHEN** 传入已取消的 context
- **THEN** 返回 codes.Canceled

#### Scenario: protocompile 编译失败
- **WHEN** fixture 的 proto 文件存在语法错误或 import 缺失
- **THEN** 返回 "protocompile failed" 错误

#### Scenario: api.lock 文件损坏
- **WHEN** api.lock 文件存在但内容为非法 YAML/JSON
- **THEN** generate 返回非 "file not found" 的读取错误

#### Scenario: glob pattern 无匹配文件
- **WHEN** import_protos.path 指向的目录不含任何 .proto 文件
- **THEN** 返回 "no .proto files matched pattern" 错误

### Requirement: fixtures 驱动

测试输入由 fixtures 目录组织，包含正向和反向测试输入。

#### Scenario: 正向 fixtures
- **WHEN** 查看 `testcase/fixtures/` 目录
- **THEN** 存在 `book/`（完整 HTTP+OpenAPI）、`simple/`（P0 纯 gRPC）、`edge/`（边界条件）三个 fixture
- **AND** 每个 fixture 包含 `api.yaml` 和 `proto/` 目录

#### Scenario: 反向 fixtures
- **WHEN** 查看 `testcase/fixtures/invalid/` 目录
- **THEN** 存在覆盖所有 13 层错误路径的 ~40 个错误配置 YAML 文件
- **AND** 每个文件对应一个明确的错误场景

### Requirement: CI 集成

在 CI 中新增 `testcase-e2e` job 自动运行所有 e2e 测试。

#### Scenario: CI 自动运行 e2e 测试
- **WHEN** push 或 PR 到 main/master 分支
- **THEN** CI 自动生成 fixtures 并运行 `testcase/positive/` 和 `testcase/negative/` 测试
- **AND** 测试失败时 CI 报错
