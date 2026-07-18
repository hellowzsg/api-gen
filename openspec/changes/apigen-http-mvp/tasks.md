## 1. YAML schema 校验扩展（HTTP 配置） <!-- 常规任务组 -->

- [x] 1.1 HTTP 配置校验（body_style/generate_openapi P1 限制 + googleapis 依赖物化校验） <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/validate_test.go`（测试 `body_style: resource` 报错、`generate_openapi: true` 报错、HTTP 启用但无 googleapis 依赖报错、HTTP 启用且有 googleapis path 依赖通过、HTTP 启用且有 googleapis git 依赖通过、HTTP 关闭时不校验 googleapis）
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidateHTTP -v`，确认失败原因是缺少功能）
  - [x] 1.1.3 写最小实现：`internal/yaml/validate.go`（新增 `validateHTTPConfig` 函数：body_style resource/generate_openapi 报错；HTTP 启用时检查 import_protos 含 googleapis 来源——path 条目路径含 `google/api` 段或 git/bsr 条目指向 googleapis 仓库）
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidateHTTP -v`，确认所有测试通过，输出干净）
  - [x] 1.1.5 重构：提取 googleapis 来源检测逻辑、统一 P1 限制错误消息格式

- [x] 1.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-mvp/specs/apigen-http.md` 和 `openspec/changes/apigen-http-mvp/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/yaml/validate.go`、`internal/yaml/validate_test.go`
    - `{BASE_SHA}` → 任务组 1 开始前的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 2. key 类型递归解析（核心新增能力） <!-- 常规任务组 -->

- [x] 2.1 key 标量叶子递归提取 <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/keyleaves_test.go`（测试简单 key 单标量叶子、复合 key 嵌套 message 深度优先序、WKT `google.protobuf.Timestamp` 视为不透明叶子、optional 标量视为普通叶子、叶子点路径格式 `org.oid`）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestKeyLeaves -v`，确认失败原因是缺少功能）
  - [x] 2.1.3 写最小实现：`internal/ir/keyleaves.go`（`ExtractKeyLeaves(keyMsg protoreflect.MessageDescriptor) ([]KeyLeaf, error)`：递归遍历字段，标量→叶子，message 非 WKT→深入，WKT（`google.protobuf.` 前缀）→不透明叶子；产出 `[]KeyLeaf{DotPath, FieldType}` 按字段声明序深度优先排序）
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestKeyLeaves -v`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：提取 WKT 判定函数、统一叶子点路径拼接逻辑

- [x] 2.2 key 递归 fail-fast 校验（repeated/map/oneof/循环引用） <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`internal/ir/keyleaves_test.go`（测试 repeated 字段 fail-fast、map 字段 fail-fast、oneof 字段 fail-fast、循环引用 A→B→A fail-fast、嵌套 message 无循环正常通过）
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/ir/ -run TestKeyLeaves_FailFast -v`，确认失败）
  - [x] 2.2.3 写最小实现：`internal/ir/keyleaves.go` 扩展 `ExtractKeyLeaves`（`IsList()`→repeated fail-fast、`IsMap()`→map fail-fast、oneof 检测→fail-fast；维护 `visited map[protoreflect.FullName]bool` 栈，遇环 fail-fast）
  - [x] 2.2.4 验证测试通过（运行：`go test ./internal/ir/ -run TestKeyLeaves_FailFast -v`，确认通过）
  - [x] 2.2.5 重构：提取校验错误消息常量、统一 fail-fast 错误类型

- [x] 2.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-mvp/specs/apigen-http.md` 和 `openspec/changes/apigen-http-mvp/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/ir/keyleaves.go`、`internal/ir/keyleaves_test.go`
    - `{BASE_SHA}` → 任务组 1 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 3. IR 扩展与 HTTP 注解生成 <!-- 常规任务组 -->

- [x] 3.1 IR 扩展（HTTPEnabled/HTTPPrefix/KeyLeaves/HTTPAnnotation） <!-- TDD 任务 -->
  - [x] 3.1.1 写失败测试：`internal/ir/builder_test.go`（测试 HTTP 关闭时 IR 无 HTTP 字段、HTTP 启用时 IR 填充 HTTPEnabled/HTTPPrefix、HTTP 启用时 EntityIR.KeyLeaves 填充、各方法 IR 携带 HTTPAnnotation{Verb,Path,Body}）
  - [x] 3.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuild_HTTP -v`，确认失败原因是缺少功能）
  - [x] 3.1.3 写最小实现：`internal/ir/builder.go`（新增 `HTTPAnnotation` 结构体；`IR` 新增 `HTTPEnabled`/`HTTPPrefix`；`EntityIR` 新增 `KeyLeaves`；各方法 IR 新增 `HTTPAnnotation` 字段；新增 `BuildOptions`/`BuildWithOptions`，HTTP 启用时调用 `ExtractKeyLeaves` 填充 KeyLeaves 并通过 `httpBuildContext` 为每个方法构造 HTTPAnnotation）
  - [x] 3.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuild_HTTP -v`，确认所有测试通过，输出干净）
  - [x] 3.1.5 重构：提取 HTTPAnnotation 构造逻辑、统一 KeyLeaves 填充时机

- [x] 3.2 HTTP 路径生成与 google.api.http 注解渲染 <!-- TDD 任务 -->
  - [x] 3.2.1 写失败测试：`internal/render/http_test.go`（测试 flat 路径拼接 `{prefix}/{service}/{collection}/{key叶子段}/{resource}`、Create=POST+body:*、Delete=DELETE 无 body、DeleteSoft=POST+body:* 路径 deleteSoft、Get=GET 无 body、BatchGet=POST+body:* 路径 batchGet、List=POST+body:* 路径 list、Update=PATCH+body:*；测试注解格式 `option (google.api.http) = { ... };`）
  - [x] 3.2.2 验证测试失败（运行：`go test ./internal/render/ -run TestRenderHTTP -v`，确认失败原因是缺少功能）
  - [x] 3.2.3 写最小实现：`internal/render/http.go`（`RenderHTTPAnnotation(ann *ir.HTTPAnnotation) string` 渲染 `option (google.api.http) = { verb: "path" [body: "body"] };`；`renderRPCWithHTTP` 辅助函数）；`internal/render/template.go` 修改：`needHTTP` 从 `irData.HTTPEnabled` 派生（当前硬编码 false）；`renderServiceRPCs` 改用 `renderRPCWithHTTP` 追加注解
  - [x] 3.2.4 验证测试通过（运行：`go test ./internal/render/ -v`，确认所有测试通过，输出干净）
  - [x] 3.2.5 重构：提取路径拼接规则表、统一注解渲染格式

- [x] 3.3 api-linter 豁免扩展（HTTP 触发） <!-- TDD 任务 -->
  - [x] 3.3.1 写失败测试：`internal/render/http_test.go` 的 `TestRenderServiceProto_HTTPAnnotation` 隐式验证（HTTP 启用时豁免含 http 系列）；补充 `TestRenderServiceProto_AllMethodVerbs` 验证全部 7 方法注解
  - [x] 3.3.2 验证测试失败（运行：`go test ./internal/render/ -run TestRenderServiceProto_HTTP -v`，确认失败原因是缺少功能）
  - [x] 3.3.3 写最小实现：`internal/render/template.go` 修改 `generateExemptions` 签名增加 `httpEnabled bool` 参数；HTTP 启用时按实际触发追加 `core::0133::http-body`（hasCreate）、`core::0231::http-body`+`core::0231::http-method`（hasBatchGet）、`core::0132::http-method`+`core::0132::http-body`（hasList）、`core::0135::http-method`+`core::0135::http-body`（hasDeleteSoft）；`internal/lint/linter.go` 的 `GenerateExemptions` 待 3.3 审查后同步（当前 render 不依赖 lint 包）
  - [x] 3.3.4 验证测试通过（运行：`go test ./internal/render/ -v`，确认通过）
  - [x] 3.3.5 重构：提取 HTTP 豁免规则表、统一豁免生成入口

- [x] 3.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-mvp/specs/apigen-http.md` 和 `openspec/changes/apigen-http-mvp/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/ir/builder.go`、`internal/ir/builder_test.go`、`internal/render/http.go`、`internal/render/http_test.go`、`internal/render/template.go`、`internal/render/template_test.go`、`internal/lint/linter.go`
    - `{BASE_SHA}` → 任务组 2 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 4. grpc-gateway 编译集成 <!-- 常规任务组 -->

- [x] 4.1 Compile 函数扩展（httpEnabled 参数 + protoc-gen-grpc-gateway 调用） <!-- TDD 任务 -->
  - [x] 4.1.1 写失败测试：`internal/build/compiler_test.go`（测试 HTTP 启用时调用 protoc-gen-grpc-gateway 生成 *.pb.gw.go、HTTP 关闭时不调用 gateway、gateway 插件参数 `paths=source_relative`、gateway 产物落盘到 goOutDir、插件未安装检测）
  - [x] 4.1.2 验证测试失败（运行：`go test ./internal/build/ -run TestCompile_HTTP -v`，确认失败原因是缺少功能）
  - [x] 4.1.3 写最小实现：`internal/build/compiler.go` 修改 `Compile` 签名增加 `httpEnabled bool` 参数；HTTP 启用时追加调用 `RunPlugin(ctx, "protoc-gen-grpc-gateway", gwReq, goOutDir)`（gwReq 参数 `paths=source_relative`）
  - [x] 4.1.4 验证测试通过（运行：`go test ./internal/build/ -v`，确认所有测试通过，输出干净）
  - [x] 4.1.5 重构：提取插件调用链构造、统一 HTTP 启用时的插件列表

- [x] 4.2 generate/build 命令集成（传递 HTTP 配置） <!-- TDD 任务 -->
  - [x] 4.2.1 写失败测试：`internal/cli/build_test.go` 现有 P0 测试验证无回归（HTTP 端到端测试在任务组 5 example 中验证）
  - [x] 4.2.2 验证测试失败（运行：`go test ./internal/cli/ -run TestBuild -v`，确认 P0 测试仍通过）
  - [x] 4.2.3 写最小实现：`internal/cli/generate.go` 新增 `buildIR(cfg, cr)` 函数（HTTP 关闭时调用 `ir.Build`，HTTP 启用时用 `cr.FindMessageDescriptor` 构建 KeyDescriptors map 并调用 `ir.BuildWithOptions`）；`internal/dep/composite.go` 新增 `FindMessageDescriptor(fqn)` 导出方法；`internal/cli/build.go` 调用 `build.Compile` 传入 `cfg.Settings.HTTP != nil && cfg.Settings.HTTP.Enable`
  - [x] 4.2.4 验证测试通过（运行：`go test ./... `，确认全量测试通过，输出干净）
  - [x] 4.2.5 重构：提取 HTTP 配置传递链、统一 generate/build 的 HTTP 配置消费

- [x] 4.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-mvp/specs/apigen-http.md` 和 `openspec/changes/apigen-http-mvp/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/build/compiler.go`、`internal/build/compiler_test.go`、`internal/cli/generate.go`、`internal/cli/build.go`、`internal/cli/build_test.go`
    - `{BASE_SHA}` → 任务组 3 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 5. example 扩展与端到端验证 <!-- 常规任务组 -->

- [x] 5.1 example 扩展（api.yaml HTTP 配置 + googleapis vendor + gw.go 产物验证） <!-- 非 TDD 任务 -->
  - [x] 5.1.1 执行变更：`examples/book/api.yaml`（增加 `settings.http.enable: true` + `prefix: /library`）；`examples/book/proto/google/api/annotations.proto` + `http.proto`（本地 vendor googleapis，annotations 导入 http）；运行 `apigen build` 生成含 HTTP 注解的 proto + `*.pb.gw.go`
  - [x] 5.1.2 验证无回归（运行：`cd examples/book && go build ./...`，确认生成的 Go 代码编译通过，含 `*.pb.gw.go`）
  - [x] 5.1.3 检查：确认生成的 proto 含 9 个 `google.api.http` 注解、import `google/api/annotations.proto`、api-linter 豁免含 http-method/http-body 系列（core::0133::http-body、core::0231::http-body、core::0231::http-method、core::0132::http-method、core::0132::http-body、core::0135::http-method、core::0135::http-body）；确认 `*.pb.gw.go` 落盘到 `generated/go/<service>/`（library_service + admin_service）

- [x] 5.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-http-mvp/specs/apigen-http.md` 和 `openspec/changes/apigen-http-mvp/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `examples/book/api.yaml`、`examples/book/proto/google/api/*.proto`、`examples/book/generated/`（生成的 proto + go 产物）
    - `{BASE_SHA}` → 任务组 4 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 6. PreCI 代码规范检查

<!-- 本项目不执行 PreCI 检查（用户明确要求），任务组 6 整体跳过 -->

- [x] 6.1 检测 preci 安装状态
  - 跳过：本项目不执行 PreCI
- [x] 6.2 检测项目是否已 preci 初始化
  - 跳过：本项目不执行 PreCI
- [x] 6.3 检测 PreCI Server 状态
  - 跳过：本项目不执行 PreCI
- [x] 6.4 执行代码规范扫描
  - 跳过：本项目不执行 PreCI
- [x] 6.5 处理扫描结果
  - 跳过：本项目不执行 PreCI

## 7. Documentation Sync (Required)

- [x] 7.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 7.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 7.3 sync proposal.md: update scope/impact if changed
- [x] 7.4 sync specs/*.md: update requirements if changed
- [x] 7.5 Final review: ensure all OpenSpec docs reflect actual implementation
