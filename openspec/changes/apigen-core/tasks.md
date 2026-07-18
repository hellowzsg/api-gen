## 1. 项目骨架与 CLI 框架 <!-- 轻量任务组：跳过独立审查，变更纳入后续任务组统一审查 -->

- [x] 1.1 初始化 Go module 与项目结构 <!-- 非 TDD 任务 -->
  - [x] 1.1.1 执行变更：`go.mod`、`cmd/apigen/main.go`、`internal/cli/`（cobra CLI 框架，注册 generate/build/dep/entity 子命令骨架）、`README.md`
  - [x] 1.1.2 验证无回归（运行：`go build ./...`，确认编译通过）
  - [x] 1.1.3 检查：确认 go.mod 含 protocompile/go-git 依赖，CLI 骨架子命令注册完整

## 2. YAML schema 解析与校验

- [x] 2.1 四段式 api.yaml schema 解析 <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/yaml/parser_test.go`（测试合法 api.yaml 解析为内部结构、缺少必填段 fail-fast、name 字段非法 fail-fast）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestParse -v`，确认失败原因是缺少功能）
  - [x] 2.1.3 写最小实现：`internal/yaml/parser.go`（定义 Config/Entity/Resource/Service 等结构体，yaml.v3 解析，schema 校验）
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestParse -v`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：整理结构体命名、提取校验函数、消除重复

- [x] 2.2 type_ 引用规则与实体引用校验 <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`internal/yaml/validate_test.go`（测试 type_ 短名/全限定名/alias 引用解析、service 引用实体越权校验、service 不声明 resources 时全量继承）
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidate -v`，确认失败）
  - [x] 2.2.3 写最小实现：`internal/yaml/validate.go`（type_ 引用解析、实体引用校验、越权校验、继承逻辑）
  - [x] 2.2.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidate -v`，确认通过）
  - [x] 2.2.5 重构：提取校验规则常量、统一错误消息格式

- [x] 2.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-core/specs/apigen.md` 和 `openspec/changes/apigen-core/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `go.mod`、`cmd/apigen/main.go`、`internal/cli/`、`internal/yaml/`
    - `{BASE_SHA}` → 任务组 1 开始前的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 3. 依赖管理（三路径）

- [x] 3.1 path 依赖解析（protocompile SourceResolver） <!-- TDD 任务 -->
  - [x] 3.1.1 写失败测试：`internal/dep/path_test.go`（测试 glob 匹配 proto 文件、SourceResolver 解析本地 proto、proto 文件不存在 fail-fast）
  - [x] 3.1.2 验证测试失败（运行：`go test ./internal/dep/ -run TestPathResolver -v`，确认失败）
  - [x] 3.1.3 写最小实现：`internal/dep/path.go`（glob 匹配、protocompile SourceResolver 封装、ImportPaths 注册）
  - [x] 3.1.4 验证测试通过（运行：`go test ./internal/dep/ -run TestPathResolver -v`，确认通过）
  - [x] 3.1.5 重构：提取 glob 工具函数、统一 ImportPaths 管理

- [x] 3.2 git 依赖拉取（go-git library）+ api.lock <!-- TDD 任务 -->
  - [x] 3.2.1 写失败测试：`internal/dep/git_test.go`（测试 branch/tag 浅克隆、commit SHA 完整 clone+checkout、api.lock 生成与读取、api.lock 一致性校验、ref 不存在 fail-fast）
  - [x] 3.2.2 验证测试失败（运行：`go test ./internal/dep/ -run TestGitResolver -v`，确认失败）
  - [x] 3.2.3 写最小实现：`internal/dep/git.go`（go-git clone、subdir 提取、api.lock 读写、resolved_commit 记录）
  - [x] 3.2.4 验证测试通过（运行：`go test ./internal/dep/ -run TestGitResolver -v`，确认通过）
  - [x] 3.2.5 重构：提取锁文件结构体、统一缓存目录管理（`.apigen_cache/git/`）

- [x] 3.3 BSR 依赖拉取（buf CLI subprocess）+ buf.yaml/buf.lock <!-- TDD 任务 -->
  - [x] 3.3.1 写失败测试：`internal/dep/bsr_test.go`（测试 buf.yaml(v2) 生成、buf dep update 调用、buf export 调用、buf 未安装检测、版本管理 APIGEN_BUF_VERSION、subprocess 参数数组形式）
  - [x] 3.3.2 验证测试失败（运行：`go test ./internal/dep/ -run TestBSRResolver -v`，确认失败）
  - [x] 3.3.3 写最小实现：`internal/dep/bsr.go`（buf CLI 检测/安装、buf.yaml 生成、subprocess 调用 buf dep update + buf export、白名单校验）
  - [x] 3.3.4 验证测试通过（运行：`go test ./internal/dep/ -run TestBSRResolver -v`，确认通过）
  - [x] 3.3.5 重构：提取 subprocess 安全调用封装、统一 buf CLI 版本管理

- [x] 3.4 复合 Resolver 组装与闭包 dry-run 校验 <!-- TDD 任务 -->
  - [x] 3.4.1 写失败测试：`internal/dep/composite_test.go`（测试 path+git+bsr 复合 Resolver 组装、import 闭包 dry-run、符号可达性校验、type_ 类型约束校验（必须 message 类型）、传递依赖递归收集）
  - [x] 3.4.2 验证测试失败（运行：`go test ./internal/dep/ -run TestComposite -v`，确认失败）
  - [x] 3.4.3 写最小实现：`internal/dep/composite.go`（CompositeResolver 组装、collectTransitiveClosure、dry-run 校验、type_ 类型约束校验）
  - [x] 3.4.4 验证测试通过（运行：`go test ./internal/dep/ -run TestComposite -v`，确认通过）
  - [x] 3.4.5 重构：提取校验规则、统一 protocompile link 结果处理

- [x] 3.5 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组（含任务组 1 轻量变更）所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-core/specs/apigen.md` 和 `openspec/changes/apigen-core/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/dep/` 全部文件
    - `{BASE_SHA}` → 任务组 1 开始前的 commit SHA（轻量组合并审查）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 4. IR 构建与 proto 生成

- [x] 4.1 IR 构建（实体→资源→方法映射，wrapper 字段号分配） <!-- TDD 任务 -->
  - [x] 4.1.1 写失败测试：`internal/ir/builder_test.go`（测试实体级 Create/Delete/DeleteSoft IR 生成、资源级 Get/BatchGet/List/Update IR 生成、wrapper 字段号从 1 连续分配、version 追加固定尾号、Create 各资源可选、Create 响应只返回 key）
  - [x] 4.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestBuilder -v`，确认失败）
  - [x] 4.1.3 写最小实现：`internal/ir/builder.go`（Entity/Resource/Method IR 结构体、wrapper 字段号分配器、RPC 命名规则 `<Verb><Entity>` / `<Verb><Entity><Resource>`）
  - [x] 4.1.4 验证测试通过（运行：`go test ./internal/ir/ -run TestBuilder -v`，确认通过）
  - [x] 4.1.5 重构：提取字段号分配规则常量、统一命名派生函数

- [x] 4.2 version 策略（STRONG/WEAK/NONE + wrapper 类型） <!-- TDD 任务 -->
  - [x] 4.2.1 写失败测试：`internal/ir/version_test.go`（测试 STRONG 标量 version、WEAK wrapper version（UInt64Value/UInt32Value/StringValue）、NONE 无 version、Get 响应回带 version、Update 请求 CAS、Update 响应 STRONG 回带新版本）
  - [x] 4.2.2 验证测试失败（运行：`go test ./internal/ir/ -run TestVersion -v`，确认失败）
  - [x] 4.2.3 写最小实现：`internal/ir/version.go`（version 策略映射、wrapper 类型选择、字段号追加逻辑）
  - [x] 4.2.4 验证测试通过（运行：`go test ./internal/ir/ -run TestVersion -v`，确认通过）
  - [x] 4.2.5 重构：提取 version 类型映射表、统一 import 生成规则（WEAK 需 wrappers.proto）

- [x] 4.3 option 注入（五档纯搬运） <!-- TDD 任务 -->
  - [x] 4.3.1 写失败测试：`internal/ir/option_test.go`（测试 field/message/rpc/service/file 五档 option 注入、option 全限定名可达性校验、target.path 存在性校验、值类型合法性（标量 + YAML map → proto text format））
  - [x] 4.3.2 验证测试失败（运行：`go test ./internal/ir/ -run TestOption -v`，确认失败）
  - [x] 4.3.3 写最小实现：`internal/ir/option.go`（五档注入逻辑、option 可达性校验、target.path 存在性校验、值类型转换）
  - [x] 4.3.4 验证测试通过（运行：`go test ./internal/ir/ -run TestOption -v`，确认通过）
  - [x] 4.3.5 重构：提取注入目标枚举、统一 proto text format 生成

- [x] 4.4 proto 模板渲染（确定性输出） <!-- TDD 任务 -->
  - [ ] 4.4.1 写失败测试：`internal/render/template_test.go`（测试 proto 文件生成、import 排序字典序、message/RPC 顺序固定、字段号固定、api-linter 豁免注释生成（按实际触发裁剪）、多次运行 bit-identical）
  - [ ] 4.4.2 验证测试失败（运行：`go test ./internal/render/ -run TestTemplate -v`，确认失败）
  - [ ] 4.4.3 写最小实现：`internal/render/template.go`（text/template 模板、import 精确生成规则、豁免注释裁剪、确定性输出保证）
  - [ ] 4.4.4 验证测试通过（运行：`go test ./internal/render/ -run TestTemplate -v`，确认通过）
  - [ ] 4.4.5 重构：提取模板片段、统一 import 生成规则表（§15.6）

- [x] 4.5 generate 命令集成 <!-- TDD 任务 -->
  - [ ] 4.5.1 写失败测试：`internal/cli/generate_test.go`（测试端到端 generate 流程：YAML 解析 → 依赖拉取 → protocompile 解析 → 语义校验 → IR 构建 → 模板渲染 → 落盘 generated/proto/<service>/<service>.proto）
  - [ ] 4.5.2 验证测试失败（运行：`go test ./internal/cli/ -run TestGenerate -v`，确认失败）
  - [ ] 4.5.3 写最小实现：`internal/cli/generate.go`（编排全流程、错误处理、落盘逻辑）
  - [ ] 4.5.4 验证测试通过（运行：`go test ./internal/cli/ -run TestGenerate -v`，确认通过）
  - [ ] 4.5.5 重构：提取流程编排、统一错误消息

- [x] 4.6 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-core/specs/apigen.md` 和 `openspec/changes/apigen-core/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/ir/`、`internal/render/`、`internal/cli/generate.go`
    - `{BASE_SHA}` → 任务组 3 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 5. 编译与后置校验

- [x] 5.1 编译 subprocess（protoc-gen-go/go-grpc） <!-- TDD 任务 -->
  - [x] 5.1.1 写失败测试：`internal/build/compiler_test.go`（测试 CodeGeneratorRequest 组装、subprocess 调用 protoc-gen-go/go-grpc、插件未安装检测、go install 自动安装、版本一致性校验、插件崩溃错误处理、gofmt 整理）
  - [x] 5.1.2 验证测试失败（运行：`go test ./internal/build/ -run TestCompiler -v`，确认失败）
  - [x] 5.1.3 写最小实现：`internal/build/compiler.go`（CodeGeneratorRequest 组装、subprocess 调用、插件管理、CodeGeneratorResponse 解析落盘、gofmt）
  - [x] 5.1.4 验证测试通过（运行：`go test ./internal/build/ -run TestCompiler -v`，确认通过）
  - [x] 5.1.5 重构：提取插件管理器、统一 subprocess 调用封装

- [x] 5.2 build 命令集成（复用 generate FDSet） <!-- TDD 任务 -->
  - [x] 5.2.1 写失败测试：`internal/cli/build_test.go`（测试 build 命令端到端：generate + 编译、复用同一 FDSet 零二次解析、产物落盘 generated/go/<service>/、可编译性形式化保证）
  - [x] 5.2.2 验证测试失败（运行：`go test ./internal/cli/ -run TestBuild -v`，确认失败）
  - [x] 5.2.3 写最小实现：`internal/cli/build.go`（编排 generate + build、FDSet 复用、产物布局）
  - [x] 5.2.4 验证测试通过（运行：`go test ./internal/cli/ -run TestBuild -v`，确认通过）
  - [x] 5.2.5 重构：提取 FDSet 复用机制、统一产物布局规则

- [x] 5.3 api-linter 豁免生成与调用 <!-- TDD 任务 -->
  - [x] 5.3.1 写失败测试：`internal/lint/linter_test.go`（测试 api-linter 豁免注释按实际触发裁剪、subprocess 调用 api-linter、豁免规则覆盖 core::0131/0133/0134/0135/0231 等、api-linter 未安装处理）
  - [x] 5.3.2 验证测试失败（运行：`go test ./internal/lint/ -run TestLinter -v`，确认失败）
  - [x] 5.3.3 写最小实现：`internal/lint/linter.go`（豁免规则映射、按触发裁剪、subprocess 调用 api-linter）
  - [x] 5.3.4 验证测试通过（运行：`go test ./internal/lint/ -run TestLinter -v`，确认通过）
  - [x] 5.3.5 重构：提取豁免规则表、统一 subprocess 调用

- [x] 5.4 buf breaking 后置校验 <!-- TDD 任务 -->
  - [x] 5.4.1 写失败测试：`internal/lint/breaking_test.go`（测试 buf breaking 调用（仅 BSR 依赖时）、buf.yaml breaking 配置、subprocess 调用 buf breaking、无 BSR 依赖时跳过）
  - [x] 5.4.2 验证测试失败（运行：`go test ./internal/lint/ -run TestBreaking -v`，确认失败）
  - [x] 5.4.3 写最小实现：`internal/lint/breaking.go`（BSR 依赖检测、buf breaking subprocess 调用、结果处理）
  - [x] 5.4.4 验证测试通过（运行：`go test ./internal/lint/ -run TestBreaking -v`，确认通过）
  - [x] 5.4.5 重构：统一后置校验编排

- [x] 5.5 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-core/specs/apigen.md` 和 `openspec/changes/apigen-core/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/build/`、`internal/cli/build.go`、`internal/lint/`
    - `{BASE_SHA}` → 任务组 4 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 6. 辅助子命令

- [ ] 6.1 dep update / dep prune / entity list 子命令 <!-- TDD 任务 -->
  - [ ] 6.1.1 写失败测试：`internal/cli/dep_test.go`（测试 dep update 强制刷新 git/BSR 依赖、dep prune 文件级反查移除未引用依赖、entity list 干跑预览实体/资源/方法清单）
  - [ ] 6.1.2 验证测试失败（运行：`go test ./internal/cli/ -run TestDep -v`，确认失败）
  - [ ] 6.1.3 写最小实现：`internal/cli/dep.go`（dep update 编排、dep prune 文件级反查、entity list 预览渲染）
  - [ ] 6.1.4 验证测试通过（运行：`go test ./internal/cli/ -run TestDep -v`，确认通过）
  - [ ] 6.1.5 重构：提取文件级反查逻辑、统一预览输出格式

- [ ] 6.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-core/specs/apigen.md` 和 `openspec/changes/apigen-core/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/cli/dep.go`、`internal/cli/entity.go`
    - `{BASE_SHA}` → 任务组 5 审查后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 7. PreCI 代码规范检查

- [x] 7.1 检测 preci 安装状态
  - 按以下优先级检测：① `~/PreCI/preci`（优先）→ ② `command -v preci`（PATH）
  - 若均未找到：执行本技能 "PreCI 代码规范检查规范" 节中的安装命令，安装完成后继续
  - 若找到：记录可用路径，直接继续
- [x] 7.2 检测项目是否已 preci 初始化
  - 检查 `.preci/`、`build.yml`、`.codecc/` 任一存在即为已初始化
  - 若未初始化：执行 `preci init`，等待完成后继续
- [x] 7.3 检测 PreCI Server 状态
  - 执行 `<preci路径> server status` 检查服务是否启动
  - 若未启动：执行 `<preci路径> server start`，等待服务启动（最多 10 秒）
  - 若启动失败且 `skip_preci: false`：暂停流程，提示用户选择操作（重试/跳过/中止），等待用户明确确认后才继续
- [x] 7.4 执行代码规范扫描
  - 依次执行两个扫描命令：
    1. `<preci路径> scan --diff`（扫描未暂存变更）
    2. `<preci路径> scan --pre-commit`（扫描已暂存变更）
  - 合并两次扫描结果，去重后统一处理
  - 仅扫描代码文件（跳过 .md/.yml/.json/.xml/.txt/.png/.jpg 等非代码文件）
- [x] 7.5 处理扫描结果
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

## 8. Documentation Sync (Required)

- [x] 8.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 8.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 8.3 sync proposal.md: update scope/impact if changed
- [x] 8.4 sync specs/*.md: update requirements if changed
- [x] 8.5 Final review: ensure all OpenSpec docs reflect actual implementation
