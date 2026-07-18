## Context

P1（apigen-http-mvp）已交付 HTTP MVP：`settings.http.enable` 总开关 + flat 路径映射（`{prefix?}/{service}/{collection}/{key叶子段...}/{resource}`）+ key 递归解析标量叶子 + grpc-gateway 生成 `*.pb.gw.go`。P1 在 `internal/yaml/validate.go` 中对 `body_style: resource` 和 `generate_openapi: true` 做了 fail-fast 报错，提示 P2 支持。

当前代码现状（P1 完成态）：
- `internal/yaml/parser.go`：`HTTPConfig` 已有 `Enable`/`Prefix`/`BodyStyle`/`GenerateOpenAPI` 字段；`ReaderDef`/`UpdateDef`/`CustomMethod` 无 `HTTP` 覆盖字段
- `internal/ir/builder.go`：`httpBuildContext` 按 P1 默认谓词生成 `HTTPAnnotation`；`CustomMethodIR` 无 `HTTPAnnotation` 字段
- `internal/render/http.go`：`RenderHTTPAnnotation` 支持 verb/path/body（body 仅 `""`/`"*"`）
- `internal/render/template.go`：custom_method 渲染无 HTTP 注解；`generateExemptions` 按 P1 规则生成
- `internal/build/compiler.go`：`Compile` 签名为 `Compile(ctx, files, fileToGenerate, goOutDir, httpEnabled)`，无 openapi 输出
- `internal/ir/keyleaves.go`：`ExtractKeyLeaves` 提取标量叶子，无 path 变量校验函数

`design-v2.md` §22 P2 行 + §8.3（逐方法覆盖与自定义方法）+ §6（http.generate_openapi）+ §21 决策 #57/#59/#62 定义了 P2 范围。

## Goals / Non-Goals

**Goals:**
1. OpenAPI v2（swagger.json）生成：`settings.http.generate_openapi: true` 时调用 `protoc-gen-openapiv2`，输出到 `settings.out.openapi`
2. 逐方法 `http` 覆盖：`reader.http` / `writer.update.http` 覆盖默认谓词/路径/body/body_style
3. custom_methods HTTP 路由：`custom_methods[].http` 支持 AIP-136 冒号语法
4. `body_style: resource`：全局或逐方法 `body_style: resource` → `body:"<资源字段名>"`

**Non-Goals:**
- JS stub 生成（protoc-gen-es / connect-es）→ 后续提案
- einride/aip-go 插件 → 后续提案
- LRO / request_id 幂等 / 变更通知 → Roadmap（design-v2.md §23）
- custom_method path 变量的 request 消息内部字段解析（真实性由 gateway 插件编译期兜底，对齐 design-v2.md §15.4）

## Decisions

### 决策 1：OpenAPI 生成

- **触发条件**：`settings.http.enable: true` && `settings.http.generate_openapi: true`
- **插件**：subprocess 调用 `protoc-gen-openapiv2`，复用 P1 已建立的 `CodeGeneratorRequest`
- **输出目录**：`settings.out.openapi`（缺省 `generated/openapi`），每 service 一个 `<service>.swagger.json`
- **插件参数**：`logtostderr=false,json_names_for_fields=false`（与 AIP 风格对齐）
- **validate.go 改动**：移除 `generate_openapi: true` 的 fail-fast（`validate.go:44-47`）

### 决策 2：逐方法 http 覆盖

- **YAML 形态**（对齐 design-v2.md §8.3）：
  ```yaml
  resources:
    - name: meta
      reader:
        list: true
        http: { verb: get, path: /library/LibraryService/book/{key.id}/metadata }
      writer:
        update:
          http: { verb: put, body_style: resource }
  ```
- **覆盖语义**：`http` 块中声明的字段（verb/path/body/body_style）覆盖默认；未声明字段继承全局默认
- **path 覆盖校验**：用户手写的 `{key.xxx.yyy}` 变量，工具校验点路径在 key 类型树中真实可达且为标量叶子或 WKT 叶子（复用 P1 `keyleaves.go` 解析逻辑）
- **body_style 覆盖**：
  - `wrapper`（默认）→ `body:"*"`
  - `resource` → `body:"<资源字段名>"`（如 `body:"meta"`），仅对 Update/Create 有意义；Get/Delete/BatchGet/List 无 body
- **用户手写 path 的 service 段重写**：用户显式覆盖 path 时原样保留，不做 `rewriteHTTPPathForService` 重写（用户自定义路径即表明要自定义）

### 决策 3：custom_methods HTTP 路由

- **YAML 形态**：
  ```yaml
  services:
    - name: LibraryService
      custom_methods:
        - name: ArchiveBook
          request: ArchiveBookRequest
          response: ArchiveBookResponse
          http:
            verb: post
            path: /library/LibraryService/book/{book_id}:archive
            body: "*"
  ```
- **path 变量校验**：custom_method 的 path 变量（如 `{book_id}`）由用户保证存在于 request 消息中，工具校验字段路径声明合法（语法合规 + 非空），不递归解析 custom request 内部（对齐 design-v2.md §15.4）
- **IR 改动**：`CustomMethodIR` 新增 `HTTPAnnotation *HTTPAnnotation` 字段

### 决策 4：body_style: resource 的 body 字段名推导

- **Update**：`body:"<资源字段名>"`，资源字段名 = `strings.ToLower(resourcePascal)`（如 `meta`、`content`），与 `buildUpdate` 中 `RequestFields[0].Name` 一致
- **Create**：`body:"<资源字段名>"` 仅当 Create 只有一个资源时有意义；多资源 Create 用 `body_style: resource` 报错（语义不明确），回退 `wrapper`
- **全局 vs 逐方法**：全局 `settings.http.body_style: resource` 对所有 Update 生效；逐方法 `http.body_style` 覆盖全局

### 决策 5：api-linter 豁免调整

- `body_style: resource` 时 Update 的 `core::0133::http-body` 豁免不触发（body 是资源字段而非 `*`），对齐 design-v2.md §16
- custom_method 的 HTTP 路由不新增豁免（AIP-136 无强制 http-method/http-body 约束）

### 决策 6：reader.http 覆盖范围（实现期修正）

实现过程中发现：`reader.http` 是 reader 块级别的覆盖，同时影响 BatchGet 和 List 会导致两者路径相同（冲突）。修正为：**reader.http 覆盖只应用于 List**（reader 的主要方法），BatchGet 保留默认 POST `/meta/batchGet` 路由。这与 design-v2.md §8.3 的示例一致（示例中 reader.http 配 list: true 使用）。

### 决策 7：service 段重写对 IsOverride 的处理（实现期修正）

原设计决策 2 规定"用户手写 path 原样保留，不做 service 段重写"。实现过程中发现：`reader.http` 在 **entity 级别**声明，所有继承该 entity 的 service 都会继承 http 覆盖。如果 IsOverride=true 时跳过 service 段重写，则多个 service 的 List 路由会完全相同（都用第一个 service 的名字），导致路由冲突。

修正为：**IsOverride=true 时仍做 service 段重写**。service 段（entity 名前的那个段）始终按当前渲染的 service 名重写，确保多 service 路由隔离。用户如需为不同 service 用完全不同的 path，需在 service 级别的 resources 中声明 http 覆盖（当前架构 service 级 resources 不支持 http 覆盖，仅 entity 级 reader.http 生效）。

## Risks / Trade-offs

1. **custom_method path 变量真实性**：工具不解析 custom request 内部，字段存在性由 gateway 插件编译期兜底（对齐 design-v2.md §15.4 残余风险，P1 已有的残余风险延续）
2. **OpenAPI 插件版本兼容**：`protoc-gen-openapiv2` 版本由 apigen `go.mod` 锁定，与 grpc-gateway 版本对齐
3. **逐方法 path 覆盖的 service 段重写**：用户显式覆盖 path 时原样保留，不做 service 段重写（用户自定义路径即表明要自定义），可能多 service 间路径重复——但这是用户显式声明的，工具不强制隔离
4. **body_style: resource + 多资源 Create**：语义不明确（哪个资源字段做 body？），报错回退 wrapper
