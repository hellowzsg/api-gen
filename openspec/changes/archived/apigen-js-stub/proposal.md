## Why

`design-v2.md` §23 长期 Roadmap #5 明确「多语言 stub：接入 connect-go、TS、Python 等插件」。当前 apigen 已支持 Go stub（`protoc-gen-go` / `protoc-gen-go-grpc`）和 HTTP gateway（`protoc-gen-grpc-gateway`）/ OpenAPI（`protoc-gen-openapiv2`），但前端/Node.js 生态的 TS stub 生成仍是空白。

`design-v2.md` §14.2 已规划 `protoc-gen-es` / `connect-es` 插件调用位置（与 go/grpc-gateway/openapiv2 并列），§4 骨架中 `settings.out.js` 和 `settings.js_repo` 字段已预留，但 `internal/build/compiler.go` 的 `Compile` 函数未调用 JS 相关插件，YAML 中也没有 `settings.plugins.js` 声明入口。

本提案落实 Roadmap #5 的 TS stub 部分：通过 `protoc-gen-es` 生成 TypeScript 消息类型与服务定义。

## What Changes

构建 **apigen JS/TS stub 生成** 能力，通过 `protoc-gen-es` 插件生成 TypeScript 产物。

### 核心能力

1. **YAML schema 扩展**：`settings.plugins.js` 字段（字符串数组，目前可选值仅 `"es"`），声明启用 JS stub 生成
2. **编译集成**：`internal/build/compiler.go` 的 `Compile` 函数新增 `jsOutDir`/`generateJS` 参数，`generateJS=true` 时 subprocess 调用 `protoc-gen-es`，插件参数 `target=ts`（生成 TypeScript）
3. **CLI build 集成**：`internal/cli/build.go` 从 `cfg.Settings.Plugins.JS` 推导 `generateJS`，从 `cfg.Settings.Out.Js`（缺省 `generated/js`）派生 `jsOutDir`

### 不在本次范围（Non-Goals）

- connect-es 客户端生成（用户明确不需要）
- 纯 JS（非 TS）输出（仅支持 `target=ts`）
- JS 包管理集成（package.json 生成、npm publish）
- 软删除增强（AIP-164）→ 用户明确不需要
- connect-go / Python 等其他语言 stub → 后续提案

## Impact

### 新增代码

- `internal/yaml/parser.go`：`Settings` 新增 `Plugins PluginsConfig` 字段；`PluginsConfig` 结构体含 `JS []string`
- `internal/build/compiler.go`：`Compile` 签名新增 `jsOutDir string` 和 `generateJS bool` 参数；`generateJS` 时调用 `RunPlugin(ctx, "protoc-gen-es", jsReq, jsOutDir)`
- `internal/cli/build.go`：推导 `generateJS`/`jsOutDir` 并传递给 `Compile`

### 依赖

- `protoc-gen-es`（subprocess，`plugins.js` 含 `"es"` 时启用；版本由 apigen `go.mod` 锁定，未装则提示 `go install`）

### 用户工作区产物

- `generated/js/<proto-relative-path>_pb.ts`（`plugins.js: [es]` 启用时生成）

### example 扩展

- `examples/book/api.yaml` 新增 `plugins.js: [es]` 示例
- `examples/book/` 新增 `e2e_js_test.go` 校验 TS 产物存在与基础内容
