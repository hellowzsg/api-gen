## Context

P0（apigen-core）已交付纯 gRPC 全能力，`design-v2.md`（1078 行）是完整设计终稿。§22 明确划分阶段：

| 阶段 | 交付 |
|------|------|
| P0 | 纯 gRPC 全能力（已交付） |
| **P1（HTTP MVP）** | `http.enable` + `http.prefix` + flat 风格 + `body:"*"` + grpc-gateway 生成 `*.pb.gw.go`；key 递归解析标量叶子做路径绑定 |
| P2 | OpenAPI v2 + 逐方法 `http` 覆盖 + custom_methods HTTP + `body_style: resource` |

本提案覆盖 P1。

### 现状（P0 已实现）

- `internal/yaml/parser.go`：已有 `HTTPConfig` 结构体（`Enable`/`Prefix`/`BodyStyle`/`GenerateOpenAPI`），但 `Enable` 缺省 false、未实际使用
- `internal/render/template.go`：`generateImports` 有 `needHTTP` 参数但调用处硬编码 `false`；`generateExemptions` 缺少 HTTP 相关豁免（`core::0231::http-body` 等）
- `internal/build/compiler.go`：`Compile` 仅调用 `protoc-gen-go` + `protoc-gen-go-grpc`，无 grpc-gateway
- `internal/dep/composite.go`：已识别 `google/api/annotations.proto` + `google/api/http.proto` 为已知路径（防线 1 已支持）
- `examples/book/api.yaml`：无 `settings.http` 配置

### 关键设计原则（继承 design-v2.md）

- **决策 A**：gRPC 为基座，HTTP 为可选叠加层（不破黑盒）
- **决策 D**：HTTP 路径风格 = flat（`{prefix?}/{service}/{collection}/{key叶子段...}/{resource}`）
- **§8.1**：wrapper 的 path 变量承载——flat 风格递归解析 key 类型树，wrapper 仍内嵌整个 `key` 字段，无需新增字段（字段号零破坏）
- **§15.4**：HTTP 绑定校验在 generate 阶段（防线 2）完成，非"信任声明 + gateway 兜底"

## Goals / Non-Goals

**Goals:**

1. `settings.http.enable: true` 启用 HTTP 叠加层，缺省 false 向后兼容
2. `settings.http.prefix` 可选 URI 前缀
3. flat 风格路径映射（`{prefix?}/{service}/{collection}/{key叶子段...}/{resource?}`）
4. key 类型递归解析标量叶子做 URL path 段绑定（WKT 不透明、repeated/map/oneof fail-fast、循环引用检测）
5. `google.api.http` 注解生成（每个 RPC）
6. 方法-谓词映射（Create=POST/Delete=DELETE/DeleteSoft=POST+body/Get=GET/BatchGet=POST+body/List=POST+body/Update=PATCH+body）
7. grpc-gateway 编译生成 `*.pb.gw.go`
8. googleapis 依赖物化校验（HTTP 启用时）
9. HTTP path 变量校验（generate 阶段，防线 2）
10. api-linter 豁免扩展（HTTP 启用时追加 http-method/http-body 系列）
11. 可编译性保证延续（防线 1-4，HTTP 启用时防线 2 追加绑定校验）

**Non-Goals:**

1. OpenAPI v2（swagger.json）生成 → P2
2. 逐方法 `http` 覆盖（`reader.http` / `writer.update.http`）→ P2
3. custom_methods HTTP 路由（AIP-136 冒号语法）→ P2
4. `body_style: resource`（body:"<资源字段>"）→ P2（P1 仅 `body_style: wrapper` 即 `body:"*"`）
5. `settings.http.generate_openapi` → P2（P1 声明该字段时报错）
6. JS stub 生成 → 后续提案
7. einride/aip-go 插件 → 后续提案

## Decisions

### D1: P1 仅支持 `body_style: wrapper`（`body:"*"`）

P1 不实现 `body_style: resource`（body:"<资源字段>"）。声明 `body_style: resource` 或 `generate_openapi: true` 时报错提示"P2 支持"。

**理由**：严格按 design-v2.md §22 划分，保持提案边界清晰；`body_style: resource` 涉及 Update 方法 body 绑定字段区分，增加复杂度，留 P2。

### D2: key 递归解析基于 protocompile linker.Message

从 protocompile link 结果获取 key 类型的 `linker.Message`，通过 `Fields()` 迭代器遍历：
- 标量字段 → 叶子（点路径如 `org.oid`）
- message 字段：全限定名前缀 `google.protobuf.` → WKT 不透明叶子；其他 → 深入递归
- `repeated`/`map`/`oneof` → fail-fast
- 循环引用：维护已访问 message 全限定名栈，遇环 fail-fast
- `optional` 标量 → 普通叶子

产出 `[]KeyLeaf{ DotPath, FieldType }`，按 proto 字段声明序深度优先排序。

**理由**：linker.Message 暴露完整字段元信息；WKT 通过全限定名前缀判定（§15.4）；标量叶子真实存在性从 link 结果校验（非"信任声明"，§15.4 最后一行）。

### D3: HTTP 路径生成独立模块 `internal/render/http.go`

路径拼接规则（§8.2）：
```
{prefix?}/{service}/{collection}/{key叶子段...}/{resource?}
```

key 叶子段格式：`{key.org.oid}/{key.org.qid}/{key.id}`（点路径前缀 `key.`）。

方法-谓词-body 映射表（§8.2 默认路径映射表）：

| 方法 | 谓词 | 默认路径 | body |
|------|------|---------|------|
| Create | POST | `/{prefix}/{service}/{collection}` | `*` |
| Delete | DELETE | `/{prefix}/{service}/{collection}/{key叶子段...}` | — |
| DeleteSoft | POST | `/{prefix}/{service}/{collection}/deleteSoft` | `*` |
| Get | GET | `/{prefix}/{service}/{collection}/{key叶子段...}/{resource}` | — |
| BatchGet | POST | `/{prefix}/{service}/{collection}/{resource}/batchGet` | `*` |
| List | POST | `/{prefix}/{service}/{collection}/{resource}/list` | `*` |
| Update | PATCH | `/{prefix}/{service}/{collection}/{key叶子段...}/{resource}` | `*` |

**理由**：BatchGet/List 用 POST + `body:"*"` 避免复合 key / 复杂 filter 的 query 序列化问题（§8.2）；DeleteSoft 用 POST + body 与 Delete 的 DELETE + path key 区分（§8.2）。

### D4: IR 扩展最小化

`internal/ir/builder.go` 新增：
- `IR.HTTPEnabled bool`
- `IR.HTTPPrefix string`
- `EntityIR.KeyLeaves []KeyLeaf`（HTTP 启用时填充）
- 方法 IR（Create/Delete/DeleteSoft/Get/BatchGet/List/Update）各新增 `HTTPAnnotation *HTTPAnnotation`

```go
type KeyLeaf struct {
    DotPath   string // e.g. "org.oid"
    FieldType string // proto scalar type name
}

type HTTPAnnotation struct {
    Verb string // GET/POST/PATCH/DELETE
    Path string // full path with {key.xxx} variables
    Body string // "", "*", or "<field>"（P1 仅 "" 或 "*"）
}
```

**理由**：HTTP 信息作为可选叠加层挂在现有方法 IR 上，不破坏 P0 结构；HTTP 关闭时所有 HTTP 字段为 nil/空。

### D5: 编译扩展——`Compile` 新增 `httpEnabled` 参数

`internal/build/compiler.go` 的 `Compile` 函数签名扩展：
```go
func Compile(ctx context.Context, files linker.Files, fileToGenerate []string, goOutDir string, httpEnabled bool) error
```

HTTP 启用时追加调用 `protoc-gen-grpc-gateway`：
- 插件参数：`paths=source_relative`
- 产物落盘到同一 `goOutDir`（与 `*.pb.go` 同目录，符合 §14.3 产物布局）
- 插件未安装则 `go install`（版本由 apigen `go.mod` 锁定）

**理由**：grpc-gateway 与 protoc-gen-go 共用同一 `CodeGeneratorRequest`（含完整依赖闭包），无需二次解析；`paths=source_relative` 保证产物落盘路径一致。

### D6: googleapis 依赖物化校验

`internal/yaml/validate.go` 新增：HTTP 启用时扫描 `import_protos`，确认至少一条目能提供 `google/api/annotations.proto`（任何来源：path/git/bsr）。校验方式：复合 Resolver 解析后检查 `google/api/annotations.proto` 是否在 linker.Files 中可达。

**理由**：§19 校验矩阵「HTTP googleapis 可达性」；fail-fast 避免到编译期才发现缺失。

### D7: api-linter 豁免扩展（按实际触发裁剪）

`internal/render/template.go` 的 `generateExemptions` 扩展（§16 完整表）：

| 豁免规则 | 触发条件 |
|---------|---------|
| `core::0133::http-body` | hasCreate && HTTP（Create 用 `body:"*"`） |
| `core::0231::http-body` | hasBatchGet && HTTP |
| `core::0231::http-method` | hasBatchGet && HTTP |
| `core::0132::http-method` | hasList && HTTP |
| `core::0132::http-body` | hasList && HTTP |
| `core::0135::http-method` | hasDeleteSoft && HTTP |
| `core::0135::http-body` | hasDeleteSoft && HTTP |

**理由**：§16「按实际触发裁剪」；HTTP 关闭时不生成这些豁免（P0 行为不变）。

### D8: 防线 2 扩展——HTTP 绑定校验

generate 阶段（防线 2 闭包 dry-run）追加：
- key 递归解析（D2）成功（无 repeated/map/oneof、无循环引用）
- 标量叶子真实存在性（从 link 结果校验）

**理由**：§15.4「叶子字段真实存在性由工具从 protocompile 解析结果中直接校验，generate 阶段即可发现错误」；防线 3 传递依赖已覆盖 annotations 链（P0 已实现）。

## Risks / Trade-offs

### R1: grpc-gateway 插件版本兼容

grpc-gateway v2 与 v3 API 有差异（v3 支持 HTTP transcoding，v2 是经典 grpc-gateway）。

**对策**：apigen `go.mod` 锁定 `grpc-ecosystem/grpc-gateway/v2` 版本；未装则 `go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@<locked-version>`。

### R2: key 递归解析对 protocompile linker API 的依赖

linker.Message 的 `Fields()` 迭代器需能区分标量/message/repeated/map/oneof。

**对策**：protocompile 基于 `protoreflect.MessageDescriptor`，字段 `Kind()`/`Cardinality()`/`IsList()`/`IsMap()` 可完整判定；WKT 通过 `FullName()` 前缀 `google.protobuf.` 判定。

### R3: googleapis 依赖来源多样化导致 example 复杂

用户可通过 path（本地 google/api/*.proto）/ git（clone googleapis subdir）/ BSR（`buf.build/googleapis/googleapis`）提供 googleapis。

**对策**：example 优先用 path 方式（本地 vendor google/api/*.proto），最简；文档说明 git/BSR 替代方式。

### R4: P1 不支持 `body_style: resource` 的前向兼容

用户若需 Update body 绑定资源字段（`body:"meta"`），P1 只能 `body:"*"`。

**对策**：有意取舍（D1）；P1 `body_style: resource` 报错提示 P2；`body:"*"` 功能完整，仅 body 粒度不同。

### R5: HTTP 启用后 api-linter 豁免增多

豁免列表变长可能掩盖真实违规。

**对策**：按实际触发裁剪（D7）；豁免均有明确偏离原因（§16 表）；HTTP 启用后 `http-*` 系列校验反而更合规（§16 备注）。

## Implementation Progress (Phase 3)

### 任务组 1：YAML schema 校验扩展（已实现）
- `internal/yaml/validate.go`：新增 `validateHTTPConfig` 方法（HTTP nil/未启用时跳过，向后兼容 P0）；P1 限制 `body_style: resource` 和 `generate_openapi: true` 报错并提示 P2；新增 `hasGoogleapisDependency` 检测 path（需含 `google/api` 段，不保守接受 broad glob）/git（URL 含 googleapis）/bsr（模块含 googleapis）三种来源
- `internal/yaml/validate_test.go`：新增 9 个测试（3 fail-fast：body_style resource、generate_openapi、无 googleapis；6 pass：path/git/bsr 来源、HTTP 关闭/nil 跳过、wrapper 默认）
- 实现决策：path 来源检测收紧为"明确含 google/api 段"，broad glob（如 `proto/**/*.proto`）不保守接受——因为 googleapis 通常单独 vendor，broad glob 可能不覆盖；protocompile link 阶段的闭包校验是权威兜底

### 任务组 2：key 类型递归解析（已实现）
- `internal/ir/keyleaves.go`：新增 `KeyLeaf{DotPath, FieldType}` 结构体和 `ExtractKeyLeaves(keyMsg protoreflect.MessageDescriptor) ([]KeyLeaf, error)` 函数
  - 递归遍历字段：标量→叶子（点路径如 `org.oid`）；message 非 WKT→深入递归；WKT（`google.protobuf.` 前缀）→不透明叶子不递归
  - fail-fast：`IsList()`（repeated）、`IsMap()`（map）、`ContainingOneof() && !HasOptionalKeyword()`（oneof，排除 proto3 optional 的 synthetic oneof）、循环引用（`visited map[protoreflect.FullName]bool` 栈 + `defer delete` 回溯）
  - `scalarKindName` 映射 protoreflect.Kind 到 proto 源类型拼写（int32/uint64/string 等）
- `internal/ir/keyleaves_test.go`：12 个测试（7 正常：简单 key、复合 key 深度优先、WKT 不透明、optional 标量、点路径格式、3 层嵌套、多标量根；5 fail-fast：repeated、map、oneof、循环引用、嵌套无环正常）
  - 测试 helper：`mkMsg`/`mkScalarField`/`mkMsgField`/`mkRepeatedScalarField`/`mkMapField`（真实 proto map entry）/`mkOptionalScalarField`/`mkOneofField`；`mkFile`/`mkFileWithImports` 用 `protodesc.NewFile` + `protoregistry.GlobalFiles` 解析 WKT；side-effect import `timestamppb` 注册 WKT

### 任务组 3：IR 扩展与 HTTP 注解生成（已实现）
- `internal/ir/builder.go`：新增 `HTTPAnnotation{Verb,Path,Body}` 结构体；`IR` 新增 `HTTPEnabled`/`HTTPPrefix`；`EntityIR` 新增 `KeyLeaves`；各方法 IR（Create/Delete/DeleteSoft/Get/BatchGet/List/Update）新增 `HTTPAnnotation *HTTPAnnotation` 字段；新增 `BuildOptions{KeyDescriptors}` 和 `BuildWithOptions(cfg, opts)`，`Build(cfg)` 保留为 `BuildWithOptions(cfg, BuildOptions{})` 的快捷方式（向后兼容 P0）
  - `httpBuildContext` 封装单实体单服务的路径生成：`keyPathSegments()` 将 KeyLeaves 转为 `{key.xxx}` 段；`joinPath()` 拼接前缀+段；`buildCreateAnnotation`/`buildDeleteAnnotation`/`buildDeleteSoftAnnotation`/`fillResourceAnnotations` 按方法类型构造 Verb/Path/Body
  - `firstServiceForEntity` 取首个引用该 entity 的 service 名（孤立 entity 用 entity 名兜底）
- `internal/render/http.go`：新增 `RenderHTTPAnnotation(ann)` 渲染 `option (google.api.http) = { verb: "path" [body: "body"] };`；`renderRPCWithHTTP` 辅助函数（nil annotation 时单行 RPC，非 nil 时多行带注解）
- `internal/render/template.go`：`needHTTP` 从 `irData.HTTPEnabled` 派生（原硬编码 false）；`renderServiceRPCs` 改用 `renderRPCWithHTTP`；`generateExemptions` 签名增加 `httpEnabled bool`，HTTP 启用时按实际触发追加 `core::0133::http-body`（hasCreate）、`core::0231::http-body`+`http-method`（hasBatchGet）、`core::0132::http-method`+`http-body`（hasList）、`core::0135::http-method`+`http-body`（hasDeleteSoft）
- `internal/ir/builder_test.go`：3 个 HTTP IR 测试（HTTPDisabled、HTTPEnabled 全方法 Verb/Body 验证、HTTPPathGeneration 路径拼接验证）
- `internal/render/http_test.go`：5 个测试（RenderHTTPAnnotation 格式 4 场景、nil 返回空、HTTPAnnotation 含 import+注解、HTTPDisabled 无注解、AllMethodVerbs 全 7 方法注解）

### 任务组 4：grpc-gateway 编译集成（已实现）
- `internal/build/compiler.go`：`Compile` 签名扩展 `httpEnabled bool` 参数；HTTP 启用时追加调用 `RunPlugin(ctx, "protoc-gen-grpc-gateway", gwReq, goOutDir)`（gwReq 参数 `paths=source_relative`，与 protoc-gen-go/go-grpc 共用同一 CodeGeneratorRequest）
- `internal/build/compiler_test.go`：新增 2 个签名检查测试（HTTPEnabledSignature、HTTPDisabledNoGateway）
- `internal/dep/composite.go`：新增 `FindMessageDescriptor(fqn) protoreflect.MessageDescriptor` 导出方法（基于现有 `findDescriptor`，返回 nil 安全处理）
- `internal/cli/generate.go`：新增 `buildIR(cfg, cr)` 函数——HTTP 关闭时调用 `ir.Build(cfg)`（P0 兼容），HTTP 启用时遍历 entities 用 `cr.FindMessageDescriptor` 构建 `KeyDescriptors map[string]protoreflect.MessageDescriptor` 并调用 `ir.BuildWithOptions(cfg, BuildOptions{KeyDescriptors})`；`runGenerate` 中 `ir.Build(cfg)` 调用替换为 `buildIR(cfg, cr)`
- `internal/cli/build.go`：`build.Compile` 调用传入 `cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable`

### 任务组 5：example 扩展与端到端验证（已实现）
- `examples/book/api.yaml`：增加 `settings.http.enable: true` + `prefix: /library`
- `examples/book/proto/google/api/annotations.proto`：本地 vendor googleapis（导入 http.proto + extend MethodOptions 注册 `http` field 72295728）
- `examples/book/proto/google/api/http.proto`：本地 vendor googleapis（定义 HttpRule/CustomHttpPattern/Http 消息）
- `examples/book/go.mod`/`go.sum`：新增 `github.com/grpc-ecosystem/grpc-gateway/v2` 和 `google.golang.org/genproto/googleapis/api/annotations` 依赖
- 端到端验证：`apigen build` 成功生成含 9 个 `google.api.http` 注解的 proto + `*.pb.gw.go`（library_service 42KB + admin_service 29KB）；`go build ./...` 编译通过
- 实现决策：`hasGoogleapisDependency` 放宽 broad glob（`**`）检测——`proto/**/*.proto` 保守接受（protocompile link 权威兜底）；`TestValidateHTTP_NoGoogleapis` 改用明确文件路径 `proto/book.proto`（无 glob、无 google/api）触发报错
