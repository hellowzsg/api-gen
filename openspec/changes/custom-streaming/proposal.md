## Why

apigen 当前所有生成的 RPC（标准方法 + `custom_methods`）均为 unary 模式，无法表达流式语义。首个重度使用方 my-wechat 项目的"通用流式导入服务"（import-enhance 提案）需要 client-streaming `ImportRows` 接口：导入脚本推送原始行、server 端做源→目标转换，unary 无法承载不定长行流。

gRPC 原生支持三种流式模式，AIP 生态（AIP-151 LRO、AIP-158 分页变体）也涉及流式语义。apigen 需要在 `custom_methods` 层补齐流式声明能力。

## What Changes

### 1. `custom_methods[].stream` 配置扩展（向后兼容）

新增 `stream` 字段，取值：

| 取值 | 含义 | proto 生成 | HTTP 支持 |
| --- | --- | --- | --- |
| `""`（默认，省略） | unary | `rpc X(Req) returns (Resp)` | 是 |
| `server` | server-streaming | `rpc X(Req) returns (stream Resp)` | 是（chunked 响应） |
| `client` | client-streaming | `rpc X(stream Req) returns (Resp)` | 否 |
| `bidi` | bidirectional-streaming | `rpc X(stream Req) returns (stream Resp)` | 否 |

- 非法取值在 validate 阶段 fail-fast
- `bidi` 是 `bidirectional` 的缩写（gRPC 生态标准缩写，`protoc-gen-go-grpc` 源码亦用此词）

### 2. HTTP 兼容性校验（编译期 fail-fast）

当 `http.enable` 为 true 且 `custom_methods[].stream` 为 `client` 或 `bidi` 时：
- `apigen generate` / `apigen build` 在 validate 阶段报错
- 错误信息指明方法名、stream 取值、不兼容原因
- `server` stream + HTTP 允许并存（grpc-gateway 自动转为 chunked 响应）

### 3. proto 渲染扩展

`renderRPCWithHTTP` 支持 stream 标记：
- unary（含 HTTP annotation）：现状不变
- server-streaming + HTTP：`rpc X(Req) returns (stream Resp) { option ... }`
- server-streaming 无 HTTP：`rpc X(Req) returns (stream Resp);`
- client/bidi（仅纯 gRPC）：`rpc X(stream Req) returns (Resp)` / `rpc X(stream Req) returns (stream Resp)`

### 4. IR 与 YAML 层贯穿

- `CustomMethod` YAML struct 新增 `Stream string`
- `CustomMethodIR` 新增 `Stream string`（取值同上，空串=unary）
- `buildService` 传递 stream 字段到 IR

### 5. 文档

- `.claude/skills/apigen-cli/SKILL.md`：方法生成映射表补充 stream 列
- `.claude/skills/apigen-cli/references/config-schema.md`：`custom_methods[].stream` 字段说明
- `.claude/skills/apigen-cli/references/examples.md`：流式示例
- `README.md` / `README_EN.md`：custom_methods 配置说明补充 stream

## Impact

### 受影响的代码

- `internal/yaml/parser.go`：`CustomMethod` 新增 `Stream string` 字段
- `internal/yaml/validate.go`：`stream` 取值校验 + HTTP 不兼容校验
- `internal/ir/builder.go`：`CustomMethodIR` 新增 `Stream` 字段，`buildService` 传递
- `internal/render/template.go` / `internal/render/http.go`：`renderRPCWithHTTP` 支持 stream 标记
- `examples/book/api.yaml` + `examples/book/generated/` + e2e 测试：新增流式 custom_method 端到端验证
- `.claude/skills/apigen-cli/` 文档更新

### 兼容性

- 默认行为不变（`stream` 省略 = unary），现有 api.yaml 与生成代码零影响
- 流式仅作用于 `custom_methods`，标准方法（Create/Delete/Get/...）保持 unary 不变
- `stream` 字段一经发布不可变更（unary ↔ streaming 切换属于 breaking change）

### 注意事项

- 仓库中存在活跃变更 `testcase-e2e-suite`（不修改 `internal/` 或 `examples/` 代码）和 `create-client-key`（已实施完毕），与本提案无冲突
- my-wechat import-enhance 提案的 `ImportRows` client-streaming 接口依赖本能力，本提案是其前置
