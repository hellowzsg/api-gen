## ADDED Requirements

### Requirement: Custom Method Stream Configuration

`custom_methods[].stream` 必须支持可选的流式模式声明，取值为 `""`（unary，默认）/ `server`（server-streaming）/ `client`（client-streaming）/ `bidi`（bidirectional-streaming）。省略 `stream` 字段等价于 `stream: ""`，保持现状 unary 行为。`bidi` 是 `bidirectional` 的标准 gRPC 缩写。

#### Scenario: 省略 stream 字段默认为 unary
- **WHEN** custom_method 声明不含 `stream` 字段
- **THEN** 生成 `rpc <Name>(<Request>) returns (<Response>)`，与现状完全一致

#### Scenario: server-streaming 模式
- **WHEN** custom_method 声明 `stream: server`
- **THEN** 生成 `rpc <Name>(<Request>) returns (stream <Response>)`

#### Scenario: client-streaming 模式
- **WHEN** custom_method 声明 `stream: client`
- **THEN** 生成 `rpc <Name>(stream <Request>) returns (<Response>)`

#### Scenario: bidirectional-streaming 模式
- **WHEN** custom_method 声明 `stream: bidi`
- **THEN** 生成 `rpc <Name>(stream <Request>) returns (stream <Response>)`

#### Scenario: 非法取值报错
- **WHEN** custom_method 声明 `stream: both`（非法值）
- **THEN** validate 阶段 fail-fast，错误信息包含方法名与合法取值（""/server/client/bidi）

### Requirement: Stream-HTTP Compatibility Validation

当 `http.enable` 为 true 时，`custom_methods[].stream` 取值为 `client` 或 `bidi` 必须在 validate 阶段报错（grpc-gateway 不支持这两种流式）。`stream: server` 与 HTTP 可并存（grpc-gateway 自动转为 chunked 响应）。`http.enable` 为 false 时，所有 stream 取值均允许。

#### Scenario: client-streaming + HTTP 启用报错
- **WHEN** `http.enable: true` 且 custom_method 声明 `stream: client`
- **THEN** validate 阶段 fail-fast，错误信息指明方法名、stream 取值、不兼容原因（仅 unary 和 server-streaming 支持 HTTP）

#### Scenario: bidi-streaming + HTTP 启用报错
- **WHEN** `http.enable: true` 且 custom_method 声明 `stream: bidi`
- **THEN** validate 阶段 fail-fast，错误信息同上

#### Scenario: server-streaming + HTTP 启用允许
- **WHEN** `http.enable: true` 且 custom_method 声明 `stream: server`
- **THEN** 校验通过，生成 `rpc <Name>(<Request>) returns (stream <Response>) { option (google.api.http) = {...}; }`

#### Scenario: 任意 stream + HTTP 禁用允许
- **WHEN** `http.enable: false`（或省略）且 custom_method 声明任意 stream 取值
- **THEN** 校验通过，生成对应流式 RPC（无 HTTP annotation）

### Requirement: Stream Field Immutability

`custom_methods[].stream` 一经发布不可变更。unary ↔ streaming 切换会改变 gRPC 方法类型（`protoc-gen-go-grpc` 生成的 interface 方法签名完全不同），属于 breaking change。

#### Scenario: 文档标注不可切换
- **WHEN** 用户查阅 config-schema.md 或 README
- **THEN** 文档明确标注"stream 字段一经发布不要切换模式"
