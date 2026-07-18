## Why

P0（apigen-core）已交付纯 gRPC 全能力：四段式 `api.yaml` → 服务层 proto → `*.pb.go` / `*_grpc.pb.go`。但实际业务中许多消费方（前端、运维脚本、第三方集成）需要 HTTP/JSON 接口，而 gRPC 无法直接满足。当前用户若要 HTTP 接口，需手写 `google.api.http` 注解并自行驱动 `protoc-gen-grpc-gateway`，与 apigen 的"声明式生成"理念冲突。

`design-v2.md` §22 明确划分 P1 阶段为「HTTP MVP」：启用 `settings.http.enable` 后，工具自动生成 `google.api.http` 注解 + `*.pb.gw.go`，用户零手写 HTTP 路由代码。本提案落实该阶段。

## What Changes

构建 **apigen HTTP MVP** 能力（P1），在 P0 纯 gRPC 基础上叠加 HTTP/JSON 转码层。

### 核心能力（P1 范围）

1. **HTTP 总开关**：`settings.http.enable: true` 生效；缺省 `false`（向后兼容 P0 纯 gRPC 用户零影响）
2. **URI 前缀**：`settings.http.prefix` 可选前缀（如 `/library`），缺省无前缀
3. **flat 风格路径映射**：`{prefix?}/{service}/{collection}/{key叶子段...}/{resource?}`，collection 取实体名原形不复数化
4. **key 类型递归解析**：递归遍历 key 类型树提取标量叶子字段，按点路径展开为 URL path 段（如 `{key.org.oid}/{key.id}`）；WKT `google.protobuf.*` 视为不透明叶子不递归进入
5. **google.api.http 注解生成**：每个 RPC 追加 `option (google.api.http) = { ... }`
6. **方法-谓词映射**：
   - Create → POST + `body:"*"`
   - Delete → DELETE（path key）
   - DeleteSoft → POST + `body:"*"`（路径 `/{collection}/deleteSoft`，key 走 body，与 Delete 区分）
   - Get → GET
   - BatchGet → POST + `body:"*"`（路径 `/{resource}/batchGet`）
   - List → POST + `body:"*"`（路径 `/{resource}/list`）
   - Update → PATCH + `body:"*"`（wrapper 风格）
7. **grpc-gateway 编译**：`apigen build` 在 HTTP 启用时追加调用 `protoc-gen-grpc-gateway`，生成 `*.pb.gw.go`
8. **googleapis 依赖物化校验**：HTTP 启用时校验 `import_protos` 中已有 googleapis（任何来源：path/git/bsr），都没有则 fail-fast
9. **HTTP path 变量校验**（generate 阶段，防线 2）：
   - repeated/map/oneof 字段 → fail-fast
   - 循环引用检测 → fail-fast
   - WKT 视为不透明叶子
   - 标量叶子真实存在性（从 protocompile link 结果校验，非"信任声明"）
10. **api-linter 豁免扩展**：HTTP 启用时按实际触发追加 `core::0231::http-body`、`core::0231::http-method`、`core::0132::http-method`、`core::0132::http-body`、`core::0135::http-method`、`core::0135::http-body`、`core::0133::http-body`

### 不在本次范围（Non-Goals，留 P2）

- OpenAPI v2（swagger.json）生成（`protoc-gen-openapiv2`）→ P2
- 逐方法 `http` 覆盖（`reader.http` / `writer.update.http` / `custom_methods[].http`）→ P2
- custom_methods HTTP 路由（AIP-136 冒号语法）→ P2
- `body_style: resource`（body:"<资源字段>"）→ P2（P1 仅支持 `body_style: wrapper` 即 `body:"*"`）
- `settings.http.generate_openapi` → P2（P1 声明该字段时报错提示 P2 支持）

## Impact

### 新增代码

- `internal/ir/keyleaves.go`：key 类型递归解析，提取标量叶子（WKT 不透明、repeated/map/oneof fail-fast、循环引用检测）
- `internal/render/http.go`：HTTP 路径拼接 + `google.api.http` 注解生成
- `internal/ir/builder.go` 扩展：IR 新增 `HTTPEnabled`/`HTTPPrefix`/`KeyLeaves`/`HTTPAnnotation`
- `internal/render/template.go` 修改：`needHTTP` 从 IR 派生（当前硬编码 false）；RPC 渲染追加 HTTP 注解；豁免扩展
- `internal/build/compiler.go` 修改：`Compile` 新增 `httpEnabled` 参数，追加 `protoc-gen-grpc-gateway` 调用
- `internal/yaml/validate.go` 修改：HTTP 启用时校验 googleapis 依赖可达；`body_style: resource` / `generate_openapi: true` 在 P1 报错
- `internal/cli/generate.go` / `build.go` 修改：传递 HTTP 配置到 IR 和编译器

### 依赖

- `protoc-gen-grpc-gateway`（subprocess，HTTP 启用时；版本由 apigen `go.mod` 锁定，未装则 `go install`）
- `google/api/annotations.proto` + `google/api/http.proto`（用户通过 path/git/bsr 任一方式提供）

### 用户工作区产物

- `generated/go/<service>/*.pb.gw.go`（HTTP 启用时追加，与 `*.pb.go` 同目录）

### example 扩展

- `examples/book/api.yaml` 增加 `settings.http.enable: true` + `prefix: /library`
- `examples/book/` 提供 googleapis 依赖（path 或 git 方式）
- `examples/book/generated/go/<service>/` 追加 `*.pb.gw.go`
