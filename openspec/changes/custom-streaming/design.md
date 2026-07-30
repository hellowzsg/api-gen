## Context

apigen 的 `custom_methods` 自 apigen-core 起仅支持 unary RPC。`renderRPCWithHTTP`（`internal/render/http.go:44`）的签名固定为 `(rpcName, reqType, respType, ann, hctx)`，无 stream 标记参数；`CustomMethod` YAML struct（`internal/yaml/parser.go:160`）和 `CustomMethodIR`（`internal/ir/builder.go:281`）均无 stream 字段。

my-wechat 项目的 import-enhance 提案（通用流式导入服务）需要 client-streaming `ImportRows` 接口，这是首个驱动场景。gRPC 原生支持 server-streaming / client-streaming / bidirectional-streaming 三种模式，AIP 生态（AIP-151 LRO、AIP-158）也涉及流式语义，apigen 需补齐。

## Goals / Non-Goals

**Goals:**
- `custom_methods[].stream` 新增配置字段，取值 `""` / `server` / `client` / `bidi`
- proto 渲染支持三种流式语法（`stream Req` / `stream Resp` / 双向）
- HTTP 兼容性编译期校验：`client` / `bidi` + HTTP 启用时报错；`server` + HTTP 允许并存
- 默认行为完全不变，向后兼容
- 仅作用于 `custom_methods`，标准方法保持 unary

**Non-Goals:**
- 不为标准方法（Create/Delete/Get/BatchGet/List/Update）增加流式模式
- 不实现 LRO（AIP-151）的完整模式（LRO 涉及 Operation message + polling，独立提案）
- 不实现 server-streaming + HTTP 的 OpenAPI 描述增强（OpenAPI v2 对 streaming 无原生支持，保持现状）
- 不实现 client/bidi + HTTP 的替代传输（如 WebSocket / SSE），纯 gRPC 限定
- 不定义流式 RPC 的服务端实现模式（属于服务端业务层）

## Decisions

### D1: 命名 `stream` 字段，取值 `server` / `client` / `bidi`

**决策**：字段名 `stream`（string），取值 `""`（unary，默认）/ `server` / `client` / `bidi`。

**理由**：
- `stream` 是 protobuf/gRPC 生态的标准关键词，proto 语法本身就是 `stream`，配置层复用避免认知负担
- `bidi` 是 `bidirectional` 的标准缩写：
  - `protoc-gen-go-grpc` 生成的 Go interface 方法名直接用 `Bidi` 前缀（如 `BidiStreamingCall`）
  - gRPC Go SDK 的 `grpc.StreamDesc` 用 `ServerStreams` / `ClientStreams` 布尔标记，`bidi = !ServerStreams && !ClientStreams` 的反义，文档中普遍用 `bidi` 指代双向
  - 比完整单词 `bidirectional` 更简洁，YAML 配置中更易读；比 `both` / `duplex` 更贴合 gRPC 术语
- `server` / `client` 与 `create.key` 的 `server` / `client` 命名风格一致（谁发起流）
- 空串默认 unary 保证存量配置零影响

**备选方案及否决**：
- `bidirectional`（完整词）：过长，YAML 中不简洁
- `both`：语义模糊（both 什么？）
- `duplex`：网络通用术语，但 gRPC 生态不常用
- `full` / `two_way`：非标准

### D2: HTTP 兼容性矩阵与 fail-fast 校验

**决策**：

| stream 取值 | http.enable=true | http.enable=false |
| --- | --- | --- |
| `""` (unary) | 允许，生成 HTTP annotation | 允许 |
| `server` | 允许，生成 HTTP annotation（grpc-gateway 转 chunked） | 允许 |
| `client` | **报错** | 允许 |
| `bidi` | **报错** | 允许 |

**理由**：
- grpc-gateway v2 对 client-streaming 和 bidirectional-streaming **不支持**（protoc-gen-grpc-gateway 遇到这两种会编译失败），必须在 apigen 层提前拦截
- server-streaming grpc-gateway 支持（响应用 `Transfer-Encoding: chunked`，每行一个 JSON 消息），可安全并存
- fail-fast 在 `validate` 阶段而非 `build` 阶段：错误信息更清晰（指向 YAML 配置位置），且 `generate` 命令也能捕获

**错误信息格式**：
```
services[%d].custom_methods[%d].stream: stream=%q is incompatible with http.enable=true (only unary and server-streaming support HTTP); method=%s
```

### D3: proto 渲染 — `renderRPCWithHTTP` 扩展签名

**决策**：`renderRPCWithHTTP` 新增 `streamMode string` 参数（取值 `""` / `"server"` / `"client"` / `"bidi"`），根据 mode 在 Req/Resp 前插入 `stream` 关键字。

**渲染规则**：
| stream | HTTP annotation | 生成形态 |
| --- | --- | --- |
| `""` | nil | `rpc X(Req) returns (Resp);` |
| `""` | 非 nil | `rpc X(Req) returns (Resp) { option ... }` |
| `server` | nil | `rpc X(Req) returns (stream Resp);` |
| `server` | 非 nil | `rpc X(Req) returns (stream Resp) { option ... }` |
| `client` | nil（强制） | `rpc X(stream Req) returns (Resp);` |
| `bidi` | nil（强制） | `rpc X(stream Req) returns (stream Resp);` |

**理由**：
- 参数名 `streamMode` 而非 `stream`：避免与 proto 关键词混淆，且 Go 变量名用完整词更清晰
- `client` / `bidi` 的 HTTP annotation 在 D2 已保证为 nil（validate 拦截），渲染层无需再判断

### D4: IR 层 — `CustomMethodIR.Stream` 字段

**决策**：`CustomMethodIR` 新增 `Stream string` 字段，`buildService` 从 `CustomMethod.Stream` 透传。

**理由**：
- 与 `CustomMethod` YAML struct 一一对应，透传逻辑最简
- 渲染层从 IR 读取 stream，不回查 YAML（保持 IR 作为单一数据源的架构约定）

### D5: 流式 custom_method 不生成 OpenAPI 描述

**决策**：`server` stream + HTTP 启用时，OpenAPI v2 中该方法的描述保持与 unary 一致的 request/response schema（不特殊标注 streaming）。

**理由**：
- OpenAPI v2 无原生 streaming 描述能力
- grpc-gateway 的 server-streaming HTTP 响应是 `chunked` + 逐行 JSON，客户端按 JSON 流解析，schema 与 unary 响应一致（只是多个）
- 特殊标注反而引入兼容性问题（标准 OpenAPI 工具不识别）

## Risks / Trade-offs

### R1: `bidi` 命名可能不直观
- **风险**：非 gRPC 背景用户不认识 `bidi`
- **缓解**：文档中首次出现时标注全称 `bidi (bidirectional)`；config-schema.md 字段说明列全取值表

### R2: 流式方法一经发布不可切换
- **风险**：unary ↔ streaming 切换会改变 gRPC 方法类型（protoc-gen-go-grpc 生成的 interface 方法签名完全不同），是 breaking change
- **缓解**：文档明确"stream 字段一经发布不要切换"；与 `create.key` 模式不可切换的约定一致

### R3: server-streaming + HTTP 的实际行为依赖 grpc-gateway
- **风险**：grpc-gateway 的 chunked 响应行为在某些 HTTP 中间件（如某些反向代理的 buffering）下可能不生效
- **缓解**：文档注明 server-streaming HTTP 依赖传输层不缓冲；纯 gRPC 调用不受影响；这是 grpc-gateway 的通用行为，非 apigen 特有问题

### R4: 与 import-enhance（my-wechat）的依赖关系
- **风险**：my-wechat import-enhance 提案的 `ImportRows` client-streaming 依赖本能力
- **缓解**：本提案是纯 apigen 侧变更，不触碰 my-wechat；import-enhance 在本提案发版后切换

## Implementation Notes

### 实施细节

1. **YAML 层**（`internal/yaml/parser.go`）：`CustomMethod` struct 新增 `Stream string` 字段（`yaml:"stream,omitempty"`），向后兼容。

2. **校验层**（`internal/yaml/validate.go`）：
   - 新增 `validateCustomMethodStream()` 方法，注册到 `ValidateReferences()` 调用链
   - `validStreamModes` map 检查取值合法性（`""` / `server` / `client` / `bidi`）
   - `streamSupportsHTTP()` 判断 HTTP 兼容性（仅 `""` 和 `server` 兼容）
   - 错误信息包含方法名、stream 取值、不兼容原因

3. **IR 层**（`internal/ir/builder.go`）：
   - `CustomMethodIR` 新增 `Stream string` 字段
   - `buildService` 中 `cmIR` 初始化时透传 `Stream: cm.Stream`

4. **渲染层**（`internal/render/http.go` + `internal/render/template.go`）：
   - `renderRPCWithHTTP` 新增 `streamMode string` 参数（插入在 `ann` 前）
   - 根据 `streamMode` 在 req/resp 前插入 `stream` 关键字（switch 处理 4 种模式）
   - `renderServiceRPCs` 中所有标准方法调用传 `""`（unary 不变）
   - `RenderServiceProto` 中 custom_methods 调用传 `cm.Stream`

5. **e2e 验证**（`examples/book/`）：
   - `proto/demo/business/book/book.proto` 新增 `StreamBookMetasRequest` / `StreamBookMetasResponse` message
   - `api.yaml` LibraryService 新增 `StreamBookMetas` custom_method（`stream: server` + HTTP annotation）
   - `e2e_http_test.go` mock server 实现 `StreamBookMetas`（`stream.Send` 两条响应）
   - `e2e_grpc_test.go` 新增子测试：客户端 `stream.Recv` 接收并验证两条响应
   - 验证了 server-streaming + HTTP 并存（`apigen build` 含 grpc-gateway 编译成功）

### 实施后确认：HTTP + server-streaming 并存

D2 中预期 server-streaming + HTTP 可并存的假设已通过实际编译验证。`apigen build`（含 `protoc-gen-grpc-gateway`）对 `rpc StreamBookMetas(StreamBookMetasRequest) returns (stream StreamBookMetasResponse) { option (google.api.http) = {...}; }` 编译成功，grpc-gateway 自动生成对应的 chunked 响应 handler。

### 实施后确认：api-linter 豁免

流式 custom_method 未触发新的 api-linter 告警。现有的 custom_method 豁免（`core::0136` 系列不适用于自定义方法名）已经覆盖。无需新增豁免。

### 未实施的部分

`client` / `bidi` 模式的 e2e 测试未加入 examples/book（因为 book 示例启用了 HTTP，无法配置 client/bidi 流式方法）。这两个模式的正确性由 `internal/yaml/validate_test.go`（HTTP 不兼容校验）和 `internal/render/template_test.go`（proto 渲染）覆盖。
