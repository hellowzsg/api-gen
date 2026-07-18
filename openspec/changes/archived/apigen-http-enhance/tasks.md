# P2: HTTP 增强 — 任务清单

## 1. YAML 解析与校验扩展

- [x] 1.1 新增 HTTPOverride 结构体并在 ReaderDef/UpdateDef/CustomMethod 上添加 HTTP 字段  <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go` — 解析含 `reader.http`/`writer.update.http`/`custom_methods[].http` 的 api.yaml，断言 HTTPOverride 字段被正确填充
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestParseHTTPOverride -v -count=1`，确认失败原因是结构体无 HTTP 字段）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go` — 新增 `HTTPOverride` 结构体（Verb/Path/Body/BodyStyle string），在 `ReaderDef`/`UpdateDef`/`CustomMethod` 上添加 `HTTP *HTTPOverride` 字段
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestParseHTTPOverride -v -count=1`，确认通过）
  - [x] 1.1.5 重构：整理代码，确认 `yaml:"http,omitempty"` tag 一致

- [x] 1.2 移除 P1 的 body_style:resource 和 generate_openapi fail-fast  <!-- 非 TDD 任务 -->
  - [x] 1.2.1 执行变更：`internal/yaml/validate.go` — 删除 `validate.go:37-47` 对 `body_style: resource` 和 `generate_openapi: true` 的 fail-fast 报错
  - [x] 1.2.2 验证无回归（运行：`go test ./internal/yaml/ -v -count=1`，确认输出干净）
  - [x] 1.2.3 检查：确认 P1 中相关的 fail-fast 测试用例已更新为允许通过

- [x] 1.3 新增逐方法 http 覆盖的 path 变量校验入口  <!-- TDD 任务 -->
  - [x] 1.3.1 写失败测试：`internal/yaml/validate_test.go` — 声明 `reader.http.path` 含 `{key.nonexistent}` 变量，断言 ValidateReferences 返回错误
  - [x] 1.3.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidateHTTPOverridePath -v -count=1`，确认失败原因是无校验逻辑）
  - [x] 1.3.3 写最小实现：`internal/yaml/validate.go` — 在 `validateHTTPConfig` 后新增 `validatePerMethodHTTPOverrides`，调用 `ir.ValidatePathVariables` 校验用户手写 path 变量（需传入 KeyDescriptors）
  - [x] 1.3.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidateHTTPOverridePath -v -count=1`，确认通过）
  - [x] 1.3.5 重构：确认校验入口与 P1 的 keyleaves 解析正确衔接

- [x] 1.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/yaml/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-enhance/specs/apigen-http-enhance.md` 和 `openspec/changes/apigen-http-enhance/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/yaml/parser.go`, `internal/yaml/validate.go`, `internal/yaml/parser_test.go`, `internal/yaml/validate_test.go`
    - `{BASE_SHA}` → 任务组开始前的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 2. IR 构建器扩展 — 逐方法 http 覆盖与 custom_method http

- [x] 2.1 httpBuildContext 支持逐方法 http 覆盖  <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go` — 构建含 `reader.http: { verb: get, path: /custom/path }` 的 IR，断言对应方法的 HTTPAnnotation 使用用户声明的 verb/path
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildPerMethodHTTPOverride -v -count=1`，确认失败原因是 builder 忽略 http 覆盖）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go` — `fillResourceAnnotations` 接受 `*apigenyaml.HTTPOverride` 参数，若非 nil 则用覆盖值替换默认 verb/path/body；`buildResource` 传递 reader/writer 的 HTTP 覆盖
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildPerMethodHTTPOverride -v -count=1`，确认通过）
  - [x] 2.1.5 重构：提取 `applyHTTPOverride(default *HTTPAnnotation, override *apigenyaml.HTTPOverride, resourceName string) *HTTPAnnotation` 辅助函数

- [x] 2.2 CustomMethodIR 新增 HTTPAnnotation 字段并传递 custom_method http  <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`internal/ir/builder_test.go` — 构建含 `custom_methods[].http` 的 IR，断言 CustomMethodIR.HTTPAnnotation 被正确填充
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuildCustomMethodHTTP -v -count=1`，确认失败原因是 CustomMethodIR 无 HTTPAnnotation 字段）
  - [x] 2.2.3 写最小实现：`internal/ir/builder.go` — `CustomMethodIR` 新增 `HTTPAnnotation *HTTPAnnotation`；`buildService` 中遍历 custom_methods，若声明 http 则构建 HTTPAnnotation（verb/path/body 直接来自用户声明，不做路径拼接）
  - [x] 2.2.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuildCustomMethodHTTP -v -count=1`，确认通过）
  - [x] 2.2.5 重构：确认 custom_method http 不走 keyleaves 校验（path 变量真实性由 gateway 兜底）

- [x] 2.3 body_style: resource 的 body 字段名推导  <!-- TDD 任务 -->
  - [x] 2.3.1 写失败测试：`internal/ir/builder_test.go` — 全局 `body_style: resource` + 资源名 `meta` 的 Update，断言 HTTPAnnotation.Body == `"meta"`；多资源 Create + body_style:resource 断言返回 error
  - [x] 2.3.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBodyStyleResource -v -count=1`，确认失败原因是 body 始终为 `"*"`）
  - [x] 2.3.3 写最小实现：`internal/ir/builder.go` — `httpBuildContext` 新增 `bodyStyle string` 和 `resourceName string`；`buildUpdateAnnotation` 根据 bodyStyle 推导 body（wrapper→`"*"`, resource→`resourceName`）；`buildCreateAnnotation` 多资源 + body_style:resource 时返回 error
  - [x] 2.3.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBodyStyleResource -v -count=1`，确认通过）
  - [x] 2.3.5 重构：确认 body_style 推导逻辑与 `buildUpdate` 中 RequestFields[0].Name 一致

- [x] 2.4 新增 ValidatePathVariables 函数校验用户手写 path 变量  <!-- TDD 任务 -->
  - [x] 2.4.1 写失败测试：`internal/ir/keyleaves_test.go` — 调用 `ValidatePathVariables("/path/{key.id}/meta", leaves)`，leaves 含 `id` 断言 nil；`ValidatePathVariables("/path/{key.bad}/meta", leaves)` 断言 error
  - [x] 2.4.2 验证测试失败（运行：`go test ./internal/ir/ -run TestValidatePathVariables -v -count=1`，确认失败原因是函数不存在）
  - [x] 2.4.3 写最小实现：`internal/ir/keyleaves.go` — 新增 `ValidatePathVariables(path string, keyLeaves []KeyLeaf) error`，用正则提取 `{key.xxx}` 变量，校验点路径在 keyLeaves 中存在
  - [x] 2.4.4 验证测试通过（运行：`go test ./internal/ir/ -run TestValidatePathVariables -v -count=1`，确认通过）
  - [x] 2.4.5 重构：确认正则兼容复合 key 点路径（`{key.org.oid}`）

- [x] 2.5 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/ir/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-enhance/specs/apigen-http-enhance.md` 和 `openspec/changes/apigen-http-enhance/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/ir/builder.go`, `internal/ir/keyleaves.go`, `internal/ir/builder_test.go`, `internal/ir/keyleaves_test.go`
    - `{BASE_SHA}` → 任务组 1 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 3. 渲染层扩展 — custom_method HTTP 注解与豁免调整

- [x] 3.1 custom_method 渲染追加 HTTP 注解  <!-- TDD 任务 -->
  - [x] 3.1.1 写失败测试：`internal/render/template_test.go` — 渲染含 HTTPAnnotation 的 CustomMethodIR，断言 proto 输出含 `option (google.api.http) = { post: "/.../{book_id}:archive" body: "*" }` 在 RPC 体内
  - [x] 3.1.2 验证测试失败（运行：`go test ./internal/render/ -run TestRenderCustomMethodHTTP -v -count=1`，确认失败原因是 custom_method 渲染无 HTTP 注解）
  - [x] 3.1.3 写最小实现：`internal/render/template.go` — `RenderServiceProto` 中 custom_method 渲染改用 `renderRPCWithHTTP`（传入 cm.HTTPAnnotation）
  - [x] 3.1.4 验证测试通过（运行：`go test ./internal/render/ -run TestRenderCustomMethodHTTP -v -count=1`，确认通过）
  - [x] 3.1.5 重构：确认 custom_method 无 HTTPAnnotation 时退化为纯 gRPC RPC（与 P1 行为一致）

- [x] 3.2 generateExemptions 按 body_style 调整  <!-- TDD 任务 -->
  - [x] 3.2.1 写失败测试：`internal/render/template_test.go` — body_style:resource 的 Create，断言输出不含 `core::0133::http-body` 豁免；body_style:wrapper 的 Create 断言含该豁免
  - [x] 3.2.2 验证测试失败（运行：`go test ./internal/render/ -run TestExemptionsBodyStyle -v -count=1`，确认失败原因是豁免未按 body_style 调整）
  - [x] 3.2.3 写最小实现：`internal/render/template.go` — `generateExemptions` 新增 `bodyStyleByMethod` 参数（或从 IR 读取），Create 的 `core::0133::http-body` 豁免仅在 body_style==wrapper 时追加
  - [x] 3.2.4 验证测试通过（运行：`go test ./internal/render/ -run TestExemptionsBodyStyle -v -count=1`，确认通过）
  - [x] 3.2.5 重构：确认豁免逻辑可扩展（后续 Update body_style:resource 也可按需调整）

- [x] 3.3 用户手写 path 不做 service 段重写  <!-- TDD 任务 -->
  - [x] 3.3.1 写失败测试：`internal/render/template_test.go` — 用户手写 `http.path: /custom/path` 的方法，渲染 AdminService 时断言 path 仍为 `/custom/path`（不被重写为 `/custom/AdminService/...`）
  - [x] 3.3.2 验证测试失败（运行：`go test ./internal/render/ -run TestUserPathNoRewrite -v -count=1`，确认失败原因是 rewriteHTTPPathForService 误改用户 path）
  - [x] 3.3.3 写最小实现：`internal/render/template.go` — IR 中标记 HTTPAnnotation 是否为用户覆盖（新增 `IsOverride bool` 字段），`rewriteHTTPPathForService` 跳过 IsOverride==true 的注解
  - [x] 3.3.4 验证测试通过（运行：`go test ./internal/render/ -run TestUserPathNoRewrite -v -count=1`，确认通过）
  - [x] 3.3.5 重构：确认 IsOverride 标记在 builder.go 中正确设置（逐方法 http 覆盖时 IsOverride=true）

- [x] 3.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/render/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-enhance/specs/apigen-http-enhance.md` 和 `openspec/changes/apigen-http-enhance/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/render/template.go`, `internal/render/http.go`, `internal/render/template_test.go`, `internal/ir/builder.go`
    - `{BASE_SHA}` → 任务组 2 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 4. OpenAPI 编译集成

- [x] 4.1 Compile 新增 openAPIOutDir 参数并调用 protoc-gen-openapiv2  <!-- TDD 任务 -->
  - [x] 4.1.1 写失败测试：`internal/build/compiler_test.go` — 调用 `Compile(ctx, files, fileToGenerate, goOutDir, openAPIOutDir, httpEnabled, generateOpenAPI)`，generateOpenAPI=true 时断言 openAPIOutDir 下生成 swagger.json 文件
  - [x] 4.1.2 验证测试失败（运行：`go test ./internal/build/ -run TestCompileOpenAPI -v -count=1`，确认失败原因是 Compile 签名无 openAPIOutDir 参数）
  - [x] 4.1.3 写最小实现：`internal/build/compiler.go` — `Compile` 签名新增 `openAPIOutDir string` 和 `generateOpenAPI bool` 参数；generateOpenAPI 时调用 `RunPlugin(ctx, "protoc-gen-openapiv2", req, openAPIOutDir)`，插件参数 `logtostderr=false,json_names_for_fields=false`
  - [x] 4.1.4 验证测试通过（运行：`go test ./internal/build/ -run TestCompileOpenAPI -v -count=1`，确认通过）
  - [x] 4.1.5 重构：确认 openAPIOutDir 在 generateOpenAPI=false 时可为空字符串

- [x] 4.2 build.go 传递 openapi 输出目录到 Compile  <!-- 非 TDD 任务 -->
  - [x] 4.2.1 执行变更：`internal/cli/build.go` — 从 `cfg.Settings.Out.OpenAPI`（缺省 `generated/openapi`）推导 openAPIOutDir；调用 `build.Compile` 时传入新参数
  - [x] 4.2.2 验证无回归（运行：`go build ./internal/cli/ && go test ./internal/cli/ -v -count=1`，确认编译通过且现有测试无回归）
  - [x] 4.2.3 检查：确认 OpenAPIOutDir 路径在 generate_openapi=false 时不创建目录

- [x] 4.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/build/ ./internal/cli/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-enhance/specs/apigen-http-enhance.md` 和 `openspec/changes/apigen-http-enhance/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/build/compiler.go`, `internal/cli/build.go`, `internal/build/compiler_test.go`
    - `{BASE_SHA}` → 任务组 3 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 5. example 扩展与端到端测试

- [x] 5.1 examples/book/api.yaml 新增 custom_method + 逐方法 http 覆盖 + generate_openapi  <!-- 非 TDD 任务 -->
  - [x] 5.1.1 执行变更：`examples/book/api.yaml` — 新增 `settings.http.generate_openapi: true` + `settings.out.openapi`；LibraryService 新增 custom_method `ArchiveBook`（含 http 冒号语法）；meta 资源新增 `reader.http` 覆盖 List 路径
  - [x] 5.1.2 验证无回归（运行：`cd examples/book && go run ../../cmd/apigen generate -f api.yaml`，确认 proto 生成成功）
  - [x] 5.1.3 检查：确认生成的 proto 含 custom_method HTTP 注解、逐方法覆盖路径、google.api.http 注解

- [x] 5.2 新增 examples/book/proto 中 ArchiveBookRequest/Response 消息定义  <!-- 非 TDD 任务 -->
  - [x] 5.2.1 执行变更：`examples/book/proto/demo/business/book/book.proto` — 新增 `ArchiveBookRequest { string book_id = 1; }` 和 `ArchiveBookResponse { bool archived = 1; }` 消息
  - [x] 5.2.2 验证无回归（运行：`cd examples/book && go run ../../cmd/apigen build -f api.yaml`，确认编译成功）
  - [x] 5.2.3 检查：确认 custom_method req/resp 类型被正确引用（全限定名）

- [x] 5.3 扩展 e2e_http_test.go 覆盖逐方法覆盖与 custom_method  <!-- TDD 任务 -->
  - [x] 5.3.1 写失败测试：`examples/book/e2e_http_test.go` — 新增子测试：逐方法覆盖的 List 路径可访问、custom_method `/{book_id}:archive` POST 可访问
  - [x] 5.3.2 验证测试失败（运行：`cd examples/book && go test -run TestE2EHTTPPerMethodOverride -v -count=1`，确认失败原因是路径不匹配或注解未生成）
  - [x] 5.3.3 写最小实现：重新生成 proto + pb.gw.go（`go run ../../cmd/apigen build -f api.yaml`），确认 HTTP 注解正确
  - [x] 5.3.4 验证测试通过（运行：`cd examples/book && go test -run TestE2EHTTPPerMethodOverride -v -count=1`，确认通过）
  - [x] 5.3.5 重构：整理测试用例命名与分组

- [x] 5.4 新增 e2e_openapi_test.go 校验 swagger.json 生成  <!-- TDD 任务 -->
  - [x] 5.4.1 写失败测试：`examples/book/e2e_openapi_test.go` — 断言 `generated/openapi/library_service.swagger.json` 文件存在且可解析为 JSON；断言 swagger 含 custom_method 路径
  - [x] 5.4.2 验证测试失败（运行：`cd examples/book && go test -run TestE2EOpenAPI -v -count=1`，确认失败原因是 swagger.json 不存在）
  - [x] 5.4.3 写最小实现：确认 `apigen build` 已启用 generate_openapi，运行 build 生成 swagger.json
  - [x] 5.4.4 验证测试通过（运行：`cd examples/book && go test -run TestE2EOpenAPI -v -count=1`，确认通过）
  - [x] 5.4.5 重构：整理断言，确认 swagger.json 路径与 `settings.out.openapi` 一致

- [x] 5.5 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `cd examples/book && go test ./... -v -count=1` 和 `go test ./... -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-enhance/specs/apigen-http-enhance.md` 和 `openspec/changes/apigen-http-enhance/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `examples/book/api.yaml`, `examples/book/proto/demo/business/book/book.proto`, `examples/book/e2e_http_test.go`, `examples/book/e2e_openapi_test.go`, `examples/book/generated/`
    - `{BASE_SHA}` → 任务组 4 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 6. PreCI 代码规范检查

- [x] 6.1 检测 preci 安装状态
  - 按以下优先级检测：① `~/PreCI/preci`（优先）→ ② `command -v preci`（PATH）
  - 若均未找到：执行本技能 "PreCI 代码规范检查规范" 节中的安装命令，安装完成后继续
  - 若找到：记录可用路径，直接继续
- [x] 6.2 检测项目是否已 preci 初始化
  - 检查 `.preci/`、`build.yml`、`.codecc/` 任一存在即为已初始化
  - 若未初始化：执行 `preci init`，等待完成后继续
- [x] 6.3 检测 PreCI Server 状态
  - 执行 `<preci路径> server status` 检查服务是否启动
  - 若未启动：执行 `<preci路径> server start`，等待服务启动（最多 10 秒）
  - 若启动失败且 `skip_preci: false`：暂停流程，提示用户选择操作（重试/跳过/中止），等待用户明确确认后才继续
- [x] 6.4 执行代码规范扫描
  - 依次执行两个扫描命令：
    1. `<preci路径> scan --diff`（扫描未暂存变更）
    2. `<preci路径> scan --pre-commit`（扫描已暂存变更）
  - 合并两次扫描结果，去重后统一处理
  - 仅扫描代码文件（跳过 .md/.yml/.json/.xml/.txt/.png/.jpg 等非代码文件）
- [x] 6.5 处理扫描结果
  - 若无告警：输出 `✅ PreCI 检查通过`，继续 Documentation Sync
  - 若有告警：自动修正（最多重试次数由配置 `max_auto_fix_rounds` 决定，默认 3 次），修正后重新扫描验证
  - **若重试用尽后仍有无法自动修正的告警且 `skip_preci: false`**：暂停流程，输出剩余问题列表及以下选项，等待用户明确确认：
    ```
    ⚠️ PreCI 检查发现无法自动修正的告警，由于配置 skip_preci: false，必须处理后才能继续。

    请选择操作：
    a. 处理 <编号> — 手动修复指定条目后继续
    b. 全部处理 — 修复所有剩余告警后继续
    c. 跳过检查 — 不修改代码，直接继续 Documentation Sync
    d. 中止 — 停止当前任务执行
    ```
    **禁止在用户未明确选择的情况下自动继续执行**

> **注**：本项目（aip-gen）配置 skip_preci=true，PreCI 检查已跳过。

## 7. Documentation Sync (Required)

- [x] 7.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 7.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 7.3 sync proposal.md: update scope/impact if changed
- [x] 7.4 sync specs/*.md: update requirements if changed
- [x] 7.5 Final review: ensure all OpenSpec docs reflect actual implementation
