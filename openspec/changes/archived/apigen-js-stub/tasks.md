# JS Stub 生成（protoc-gen-es）— 任务清单

## 1. YAML schema 扩展 — settings.plugins.js 声明与校验

- [x] 1.1 新增 PluginsConfig 结构体并在 Settings 上添加 Plugins 字段  <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go` — 解析含 `settings.plugins.js: [es]` 的 api.yaml，断言 `cfg.Settings.Plugins.JS == []string{"es"}`；解析省略 plugins 的 api.yaml，断言 `cfg.Settings.Plugins.JS == nil`
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestParsePluginsJS -v -count=1`，确认失败原因是 Settings 无 Plugins 字段）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go` — 新增 `PluginsConfig` 结构体（含 `JS []string` 字段，`yaml:"js,omitempty"`），在 `Settings` 上添加 `Plugins PluginsConfig` 字段（`yaml:"plugins,omitempty"`）
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestParsePluginsJS -v -count=1`，确认通过）
  - [x] 1.1.5 重构：整理代码，确认 `yaml:"plugins,omitempty"` tag 一致

- [x] 1.2 校验 plugins.js 元素只能是 "es"  <!-- TDD 任务 -->
  - [x] 1.2.1 写失败测试：`internal/yaml/validate_test.go`（若不存在则新建） — `plugins.js: [es]` 断言 nil；`plugins.js: [connect-es]` 断言 error 含 "unknown JS plugin"；`plugins.js: [es, connect-es]` 断言 error
  - [x] 1.2.2 验证测试失败（运行：`go test ./internal/yaml/ -run TestValidatePluginsJS -v -count=1`，确认失败原因是无校验逻辑）
  - [x] 1.2.3 写最小实现：`internal/yaml/validate.go` — 在 `validate()` 中新增 `validatePlugins`，遍历 `cfg.Settings.Plugins.JS`，元素非 `"es"` 则 fail-fast
  - [x] 1.2.4 验证测试通过（运行：`go test ./internal/yaml/ -run TestValidatePluginsJS -v -count=1`，确认通过）
  - [x] 1.2.5 重构：确认校验入口与现有 validate 流程衔接

- [x] 1.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/yaml/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-js-stub/specs/apigen-js-stub.md` 和 `openspec/changes/apigen-js-stub/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/yaml/parser.go`, `internal/yaml/validate.go`, `internal/yaml/parser_test.go`, `internal/yaml/validate_test.go`
    - `{BASE_SHA}` → 任务组开始前的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 2. 编译集成 — Compile 调用 protoc-gen-es + CLI build 传递参数

- [x] 2.1 Compile 新增 jsOutDir/generateJS 参数并调用 protoc-gen-es  <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/build/compiler_test.go` — 调用 `Compile(ctx, files, fileToGenerate, goOutDir, openAPIOutDir, jsOutDir, httpEnabled, generateOpenAPI, generateJS)`，generateJS=true 时断言 jsOutDir 下生成 `*_pb.ts` 文件
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/build/ -run TestCompileJS -v -count=1`，确认失败原因是 Compile 签名无 jsOutDir/generateJS 参数）
  - [x] 2.1.3 写最小实现：`internal/build/compiler.go` — `Compile` 签名新增 `jsOutDir string` 和 `generateJS bool` 参数；generateJS && jsOutDir != "" 时 `os.MkdirAll(jsOutDir)`，clone req 设置 `parameter = "target=ts"`，调用 `RunPlugin(ctx, "protoc-gen-es", jsReq, jsOutDir)`
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/build/ -run TestCompileJS -v -count=1`，确认通过）
  - [x] 2.1.5 重构：确认 jsOutDir 在 generateJS=false 时可为空字符串，不创建目录

- [x] 2.2 build.go 推导 generateJS/jsOutDir 并传递给 Compile  <!-- 非 TDD 任务 -->
  - [x] 2.2.1 执行变更：`internal/cli/build.go` — 从 `cfg.Settings.Plugins.JS` 判断是否含 `"es"` → `generateJS`；从 `cfg.Settings.Out.Js`（缺省 `generated/js`）推导 `jsOutDir`；调用 `build.Compile` 时传入新参数
  - [x] 2.2.2 验证无回归（运行：`go build ./internal/cli/ && go test ./internal/cli/ -v -count=1`，确认编译通过且现有测试无回归）
  - [x] 2.2.3 检查：确认 generateJS=false 时不创建 js 目录、不调用 protoc-gen-es

- [x] 2.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `go test ./internal/build/ ./internal/cli/ -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-js-stub/specs/apigen-js-stub.md` 和 `openspec/changes/apigen-js-stub/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `internal/build/compiler.go`, `internal/cli/build.go`, `internal/build/compiler_test.go`
    - `{BASE_SHA}` → 任务组 1 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 3. example 扩展与端到端测试

- [x] 3.1 examples/book/api.yaml 新增 plugins.js: [es] 声明  <!-- 非 TDD 任务 -->
  - [x] 3.1.1 执行变更：`examples/book/api.yaml` — `settings` 下新增 `plugins: { js: [es] }`
  - [x] 3.1.2 验证无回归（运行：`cd examples/book && go run ../../cmd/apigen generate -f api.yaml`，确认 proto 生成成功且无校验报错）
  - [x] 3.1.3 检查：确认 api.yaml 解析通过，`plugins.js` 字段被正确识别

- [x] 3.2 新增 e2e_js_test.go 校验 TS 产物生成  <!-- TDD 任务 -->
  - [x] 3.2.1 写失败测试：`examples/book/e2e_js_test.go` — 断言 `generated/js/` 目录下存在 `*_pb.ts` 文件；至少一个文件内容含 `LibraryService` 字符串（service 定义）；断言文件可被基础文本解析（含 `import` 或 `export` 关键字，表明是有效 TS 模块）
  - [x] 3.2.2 验证测试失败（运行：`cd examples/book && go test -run TestE2EJSStub -v -count=1`，确认失败原因是 `generated/js/` 目录或 `*_pb.ts` 文件不存在）
  - [x] 3.2.3 写最小实现：运行 `cd examples/book && go run ../../cmd/apigen build -f api.yaml` 生成 TS 产物（前提：protoc-gen-es 已安装；若未安装则先 `go install github.com/bufbuild/protoc-gen-es/cmd/protoc-gen-es@latest`）
  - [x] 3.2.4 验证测试通过（运行：`cd examples/book && go test -run TestE2EJSStub -v -count=1`，确认通过）
  - [x] 3.2.5 重构：整理断言，确认 TS 产物路径与 `settings.out.js` 一致

- [x] 3.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行 `cd examples/book && go test ./... -v -count=1` 和 `go test ./... -v -count=1`，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-js-stub/specs/apigen-js-stub.md` 和 `openspec/changes/apigen-js-stub/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → `examples/book/api.yaml`, `examples/book/e2e_js_test.go`, `examples/book/generated/js/`
    - `{BASE_SHA}` → 任务组 2 审查通过后的 commit SHA
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入
  - 若仅有 Minor 或无问题：自动继续下一任务组

## 4. PreCI 代码规范检查

- [x] 4.1 检测 preci 安装状态
  - 按以下优先级检测：① `~/PreCI/preci`（优先）→ ② `command -v preci`（PATH）
  - 若均未找到：执行本技能 "PreCI 代码规范检查规范" 节中的安装命令，安装完成后继续
  - 若找到：记录可用路径，直接继续
- [x] 4.2 检测项目是否已 preci 初始化
  - 检查 `.preci/`、`build.yml`、`.codecc/` 任一存在即为已初始化
  - 若未初始化：执行 `preci init`，等待完成后继续
- [x] 4.3 检测 PreCI Server 状态
  - 执行 `<preci路径> server status` 检查服务是否启动
  - 若未启动：执行 `<preci路径> server start`，等待服务启动（最多 10 秒）
  - 若启动失败且 `skip_preci: false`：暂停流程，提示用户选择操作（重试/跳过/中止），等待用户明确确认后才继续
- [x] 4.4 执行代码规范扫描
  - 依次执行两个扫描命令：
    1. `<preci路径> scan --diff`（扫描未暂存变更）
    2. `<preci路径> scan --pre-commit`（扫描已暂存变更）
  - 合并两次扫描结果，去重后统一处理
  - 仅扫描代码文件（跳过 .md/.yml/.json/.xml/.txt/.png/.jpg 等非代码文件）
- [x] 4.5 处理扫描结果
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

## 5. Documentation Sync (Required)

- [x] 5.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 5.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 5.3 sync proposal.md: update scope/impact if changed
- [x] 5.4 sync specs/*.md: update requirements if changed
- [x] 5.5 Final review: ensure all OpenSpec docs reflect actual implementation
