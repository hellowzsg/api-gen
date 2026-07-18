## Why

P1（apigen-http-mvp）已交付 HTTP MVP：`settings.http.enable` 总开关 + flat 路径映射 + key 递归解析 + grpc-gateway 生成 `*.pb.gw.go`。但以下能力在 P1 中被显式排除（Non-Goals，fail-fast 报错提示 P2 支持）：

1. OpenAPI v2（swagger.json）生成 — 前端/运维/第三方集成需要机器可读的 API 文档
2. 逐方法 `http` 覆盖 — 实际业务中部分方法需要非默认谓词/路径（如 List 用 GET 而非 POST）
3. custom_methods HTTP 路由 — AIP-136 冒号语法（如 `/{book_id}:archive`）是自定义方法的标准 REST 路由
4. `body_style: resource` — Update 方法 body 绑定到资源字段而非整个 wrapper，更贴合 REST 语义

`design-v2.md` §22 明确划分 P2 阶段为「HTTP 增强」，本提案落实该阶段。

## What Changes

构建 **apigen HTTP 增强** 能力（P2），在 P1 HTTP MVP 基础上扩展 HTTP/JSON 转码层。

### 核心能力（P2 范围）

1. **OpenAPI v2（swagger.json）生成**：`settings.http.generate_openapi: true` 时调用 `protoc-gen-openapiv2`，输出到 `settings.out.openapi`（缺省 `generated/openapi`），每 service 一个 `<service>.swagger.json`
2. **逐方法 `http` 覆盖**：`reader.http` / `writer.update.http` 可覆盖默认谓词（verb）、路径（path）、body、`body_style`；未指定字段继承全局 `settings.http` 默认值
3. **custom_methods HTTP 路由**：`custom_methods[].http` 支持 AIP-136 冒号语法（如 `/{book_id}:archive`）
4. **`body_style: resource`**：`settings.http.body_style: resource`（全局）或逐方法 `http.body_style: resource` → `body:"<资源字段名>"`（如 `body:"meta"`），仅对 Update/Create 有意义

### path 覆盖校验

- 用户手写的 `{key.xxx.yyy}` 变量，工具校验点路径在 key 类型树中真实可达且为标量叶子或 WKT 叶子（复用 P1 `keyleaves.go` 解析逻辑）
- custom_method 的 path 变量（如 `{book_id}`）由用户保证存在于 request 消息中，工具校验字段路径声明合法（语法合规 + 非空），不递归解析 custom request 内部

### api-linter 豁免调整

- `body_style: resource` 时 Update 的 `core::0133::http-body` 豁免不触发（body 是资源字段而非 `*`）
- custom_method 的 HTTP 路由不新增豁免（AIP-136 无强制 http-method/http-body 约束）

### 不在本次范围（Non-Goals）

- JS stub 生成（protoc-gen-es / connect-es）→ 后续提案
- einride/aip-go 插件 → 后续提案
- LRO / request_id 幂等 / 变更通知 → Roadmap（design-v2.md §23）

## Impact

### 新增代码

- `internal/yaml/parser.go`：`ReaderDef`/`UpdateDef`/`CustomMethod` 新增 `HTTP *HTTPOverride` 字段；`HTTPOverride` 结构体（Verb/Path/Body/BodyStyle）
- `internal/yaml/validate.go`：移除 `body_style: resource` 和 `generate_openapi: true` 的 fail-fast；新增逐方法 http 覆盖的 path 变量校验调用入口
- `internal/ir/builder.go`：`httpBuildContext` 支持逐方法覆盖；`CustomMethodIR` 新增 `HTTPAnnotation` 字段；`buildService` 传递 custom_method http
- `internal/ir/keyleaves.go`：新增 `ValidatePathVariables(path string, keyLeaves []KeyLeaf) error` 校验用户手写 path 变量可达
- `internal/render/template.go`：custom_method 渲染追加 HTTP 注解；`generateExemptions` 按 body_style 调整
- `internal/render/http.go`：`RenderHTTPAnnotation` 支持 `body_style: resource` 推导 body 字段名
- `internal/build/compiler.go`：`Compile` 新增 `openAPIOutDir string` 参数，`generate_openapi` 时调用 `protoc-gen-openapiv2`
- `internal/cli/build.go`：传递 `openapi` 输出目录到 `Compile`

### 依赖

- `protoc-gen-openapiv2`（subprocess，HTTP + generate_openapi 启用时；版本由 apigen `go.mod` 锁定，未装则 `go install`）
- 复用 P1 的 `protoc-gen-grpc-gateway`（无新增）

### 用户工作区产物

- `generated/openapi/<service>.swagger.json`（HTTP + generate_openapi 启用时生成）
- `generated/go/<service>/*.pb.gw.go`（P1 已有，无变化）

### example 扩展

- `examples/book/api.yaml` 新增 custom_method + 逐方法 http 覆盖 + `generate_openapi: true` 示例
- `examples/book/` 新增 `e2e_openapi_test.go` + 扩展 `e2e_http_test.go` 覆盖逐方法覆盖与 custom_method
