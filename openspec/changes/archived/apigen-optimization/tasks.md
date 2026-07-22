# apigen-optimization Tasks

## 1. Pipeline 复用重构（消除 build 双重执行 + 多 path 修复）

- [x] 1.1 抽取 Pipeline 结构并改造 runGenerate  <!-- TDD 任务 -->
  - [x] 1.1.1 写失败测试：`internal/cli/pipeline_test.go`（Prepare 返回 Config/ImportPaths/PathResolvers/Resolver/IR；多 path import 时 PathResolvers 含全部条目；api.lock 非法 YAML 时返回错误）
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/cli/ -run TestPrepare -count=1`，确认失败原因是 Pipeline/Prepare 不存在）
  - [x] 1.1.3 写最小实现：`internal/cli/pipeline.go`（Pipeline 结构 + Prepare：parseConfig → ValidateReferences → resolveDependencies → 收集全部 path resolver → CompositeResolver.Resolve → validateTypeReferences → buildIR → TypeImportPaths → ValidateAllOptions）；`internal/cli/generate.go` 的 runGenerate 改为 Prepare + 渲染 staging + commitDir
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/cli/ -count=1`，确认所有测试通过，输出干净）
  - [x] 1.1.5 重构：删除 runGenerate 中被 Prepare 吸收的私有步骤函数残留；保持错误包装文案与现状一致

- [x] 1.2 runBuild 复用 Pipeline  <!-- TDD 任务 -->
  - [x] 1.2.1 写失败测试：`internal/cli/pipeline_test.go` 追加（runBuild 全流程仅调用一次 Prepare——通过计数 fake 或拆分纯函数验证；多 path 场景 compile 文件集包含全部 path 文件）
  - [x] 1.2.2 验证测试失败（运行：`go test ./internal/cli/ -run TestRunBuild -count=1`，确认失败原因）
  - [x] 1.2.3 写最小实现：`internal/cli/build.go`（runBuild = Prepare + 渲染提交 + collectSeedFiles + fileToGenerate 组装 + build.Compile；删除重复 parseConfig/resolveDependencies/Glob/Resolve；`_ =` 吞错误全部改为显式返回）
  - [x] 1.2.4 验证测试通过（运行：`go test ./internal/cli/ -count=1 && go build ./...`，输出干净）
  - [x] 1.2.5 重构：整理 fileToGenerate 组装逻辑为独立函数，与 generate.go 中的相对路径换算去重

- [x] 1.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/apigen-optimization/specs/*.md` 和 `openspec/changes/apigen-optimization/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 2. 依赖解析统一与 BSR 批量化

- [x] 2.1 统一 dep.Resolver 接口  <!-- TDD 任务 -->
  - [x] 2.1.1 写失败测试：`internal/dep/resolver_test.go`（path/git/bsr 三实现均满足 Resolver 接口断言；resolveDependencies 按声明顺序聚合 importPaths）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/dep/ -run TestResolver -count=1`，确认失败原因是接口不存在）
  - [x] 2.1.3 写最小实现：`internal/dep/resolver.go`（Resolver 接口定义 + 三实现适配）；`internal/cli/pipeline.go` 的 resolveDependencies 改遍历 []Resolver
  - [x] 2.1.4 验证测试通过（运行：`go test ./internal/dep/ ./internal/cli/ -count=1`，输出干净）
  - [x] 2.1.5 重构：消除适配层的重复错误包装

- [x] 2.2 BSR 批量处理 + export 缓存 + 临时工作目录  <!-- TDD 任务 -->
  - [x] 2.2.1 写失败测试：`internal/dep/bsr_test.go` 追加（多 BSR 依赖单次 buf dep update——通过 fake buf 脚本记录调用次数；export 缓存命中跳过；用户工作区不出现 buf.yaml/buf.lock；version 字段拼入 export ref；ImportPaths 只含本次声明模块）
  - [x] 2.2.2 验证测试失败（运行：`go test ./internal/dep/ -run TestBSR -count=1`，确认失败原因）
  - [x] 2.2.3 写最小实现：`internal/dep/bsr.go`（临时目录生成 buf.yaml；解析 buf.lock 取 digest 作缓存 key；export 缓存命中跳过；ImportPaths 按声明模块返回）；`internal/cli/pipeline.go` 聚合全部 BSR 依赖为单 resolver
  - [x] 2.2.4 验证测试通过（运行：`go test ./internal/dep/ ./internal/cli/ -count=1`，输出干净）
  - [x] 2.2.5 重构：整理缓存目录布局常量，与 git 模块的 module-proxy 布局对齐

- [x] 2.3 dep prune 真实实现  <!-- TDD 任务 -->
  - [x] 2.3.1 写失败测试：`internal/cli/dep_test.go` 追加（未被引用的 git 依赖被移除；无法判定时保留；全部引用时 lock 不变）
  - [x] 2.3.2 验证测试失败（运行：`go test ./internal/cli/ -run TestDepPrune -count=1`，确认失败原因是 isDepReferenced 恒 true）
  - [x] 2.3.3 写最小实现：`internal/cli/dep.go`（基于 Prepare 的 BuildTypeImportPaths + 各 git dep 克隆目录前缀匹配实现归属判定；删除恒 true 桩）
  - [x] 2.3.4 验证测试通过（运行：`go test ./internal/cli/ -count=1`，输出干净）
  - [x] 2.3.5 重构：无

- [x] 2.4 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review（占位符映射同 1.3，{BASE_SHA} 为本任务组开始前 SHA）
  - Critical/Important 处理流程同 1.3；Minor/无问题自动继续

## 3. 插件注册表与并行编译

- [x] 3.1 PluginSpec 声明式注册 + errgroup 并行 + 浅拷贝  <!-- TDD 任务 -->
  - [x] 3.1.1 写失败测试：`internal/build/compiler_test.go` 追加（Compile 接受 []PluginSpec 并全部执行——fake 插件脚本记录调用；单插件失败错误含插件名；`go test -race` 无数据竞争）
  - [x] 3.1.2 验证测试失败（运行：`go test ./internal/build/ -run TestCompileSpec -count=1`，确认失败原因是 PluginSpec 不存在）
  - [x] 3.1.3 写最小实现：`internal/build/compiler.go`（PluginSpec 类型；Compile(ctx, files, fileToGenerate, specs) 签名重构；errgroup 并行；浅拷贝替换 proto.Clone）；`internal/cli/build.go` 按配置组装 specs；`go.mod` 将 golang.org/x/sync 提升为直接依赖
  - [x] 3.1.4 验证测试通过（运行：`go test -race ./internal/build/ ./internal/cli/ -count=1`，输出干净）
  - [x] 3.1.5 重构：提取 spec 组装为 `pluginSpecsForConfig(cfg)` 便于测试

- [x] 3.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review（占位符映射同 1.3，{BASE_SHA} 为本任务组开始前 SHA）
  - Critical/Important 处理流程同 1.3；Minor/无问题自动继续

## 4. HTTP 路径结构化（消除启发式重写）

- [x] 4.1 IR HTTPAnnotation 结构化 + 渲染按 service 拼接  <!-- TDD 任务 -->
  - [x] 4.1.1 写失败测试：`internal/ir/builder_test.go` + `internal/render/template_test.go` 改造（IR 注解携带结构化段而非最终路径字符串；同一实体挂两个 service 时渲染路径各自独立；override 路径原样保留不重写；prefix/复合 key 场景与改造前 golden 一致）
  - [x] 4.1.2 验证测试失败（运行：`go test ./internal/ir/ ./internal/render/ -count=1`，确认失败原因）
  - [x] 4.1.3 写最小实现：`internal/ir/builder.go`（HTTPAnnotation 结构化；删除 firstServiceForEntity、buildCreateAnnotation）；`internal/render/template.go`（路径拼接函数 joinHTTPPath(prefix, svc, entity, keyLeaves, resource, suffix)；删除 replaceServiceSegment/rewriteHTTPPathForService）
  - [x] 4.1.4 验证测试通过（运行：`go test ./internal/ir/ ./internal/render/ -count=1`，输出干净；追加 golden：`go run ./cmd/apigen build -f examples/book/api.yaml && git diff --exit-code examples/book/`，差异仅限 override 不重写的预期项并人工核对）
  - [x] 4.1.5 重构：收敛路径拼接的字符串处理，消除模板内散落 fmt.Sprintf

- [x] 4.2 ValidatePathVariables 接入 override 校验  <!-- TDD 任务 -->
  - [x] 4.2.1 写失败测试：`internal/ir/keyleaves_test.go` 追加（override path 含非法 `{key.*}` 变量时 BuildWithOptions 返回错误；合法变量通过；非 key 变量不校验）
  - [x] 4.2.2 验证测试失败（运行：`go test ./internal/ir/ -run TestValidatePathVariables -count=1`，确认失败原因是校验未接入）
  - [x] 4.2.3 写最小实现：`internal/ir/builder.go`（applyOverride 设置 OverridePath 时调用 ValidatePathVariables 对照 keyLeaves）
  - [x] 4.2.4 验证测试通过（运行：`go test ./internal/ir/ -count=1`，输出干净）
  - [x] 4.2.5 重构：删除 `internal/yaml/validate.go:177` 的 P2 遗留 TODO 注释

- [x] 4.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review（占位符映射同 1.3，{BASE_SHA} 为本任务组开始前 SHA）
  - Critical/Important 处理流程同 1.3；Minor/无问题自动继续

## 5. 正确性修复与死代码清除

- [x] 5.1 FormatOptionValue map key 排序（可重现生成）  <!-- TDD 任务 -->
  - [x] 5.1.1 写失败测试：`internal/ir/option_test.go` 追加（同一 map 多次格式化输出一致且 key 字典序；嵌套 map 有序）
  - [x] 5.1.2 验证测试失败（运行：`go test ./internal/ir/ -run TestFormatOptionValue -count=20`，确认乱序失败）
  - [x] 5.1.3 写最小实现：`internal/ir/option.go`（map 分支 sort.Strings(keys) 后拼接）
  - [x] 5.1.4 验证测试通过（运行：`go test ./internal/ir/ -count=1`，输出干净）
  - [x] 5.1.5 重构：无

- [x] 5.2 CompositeResolver FQN 索引 + entity list lenient 模式  <!-- TDD 任务 -->
  - [x] 5.2.1 写失败测试：`internal/dep/composite_test.go` 追加（索引查找结果与线性扫描一致——基准断言）；`internal/cli/dep_test.go` 追加（HTTP 启用时 entity list 成功输出）
  - [x] 5.2.2 验证测试失败（运行：`go test ./internal/dep/ ./internal/cli/ -count=1`，确认失败原因）
  - [x] 5.2.3 写最小实现：`internal/dep/composite.go`（Resolve 末尾构建 descByFQN；findDescriptor 改 map）；`internal/ir/builder.go`（BuildOptions.LenientHTTP）；`internal/cli/dep.go`（entity list 传 LenientHTTP）
  - [x] 5.2.4 验证测试通过（运行：`go test ./internal/dep/ ./internal/ir/ ./internal/cli/ -count=1`，输出干净）
  - [x] 5.2.5 重构：BuildTypeImportPaths 复用同一索引

- [x] 5.3 删除 internal/lint 整包与 composite 死方法  <!-- 非 TDD 任务 -->
  - [x] 5.3.1 执行变更：删除 `internal/lint/`（linter.go、breaking.go、linter_test.go、breaking_test.go）；删除 `internal/dep/composite.go` 的 DryRunClosure/CollectTransitiveClosure/ResolveWithFiles/AddImportPaths/CheckSymbolReachable 及 `composite_test.go` 中对应用例；删除 `internal/ir/builder.go` 的 buildCreateAnnotation
  - [x] 5.3.2 验证无回归（运行：`go build ./... && go vet ./... && go test ./... -count=1`，输出干净）
  - [x] 5.3.3 检查：`search_content` 确认无残留引用（lint\.、DryRunClosure、CollectTransitiveClosure、ResolveWithFiles、AddImportPaths、CheckSymbolReachable、buildCreateAnnotation）

- [x] 5.4 命名工具归一与杂项清理  <!-- 非 TDD 任务 -->
  - [x] 5.4.1 执行变更：`internal/ir/builder.go` toSnakeCase 导出为 ToSnakeCase；`internal/cli/generate.go` 删除本地 toSnakeCase 改用 ir.ToSnakeCase
  - [x] 5.4.2 验证无回归（运行：`go build ./... && go test ./... -count=1`，输出干净）
  - [x] 5.4.3 检查：确认 cli 包内无重复命名转换函数；examples 产物 diff 为空

- [x] 5.5 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review（占位符映射同 1.3，{BASE_SHA} 为本任务组开始前 SHA）
  - Critical/Important 处理流程同 1.3；Minor/无问题自动继续

## 6. 关键节点日志（slog）

- [x] 6.1 全局 verbose flag + Pipeline/dep/build 关键节点插桩  <!-- TDD 任务 -->
  - [x] 6.1.1 写失败测试：`internal/cli/logging_test.go`（替换 slog default 为 buffer handler：`-v` 下断言 config parsed / dep fetch done（cache_hit）/ protos resolved / plugin done 日志出现；默认级别断言无输出）
  - [x] 6.1.2 验证测试失败（运行：`go test ./internal/cli/ -run TestLogging -count=1`，确认失败原因是日志不存在）
  - [x] 6.1.3 写最小实现：`internal/cli/root.go`（持久化 `-v/--verbose` flag + PersistentPreRun 按 flag/APIGEN_LOG_LEVEL 初始化 slog TextHandler → stderr）；`internal/cli/pipeline.go`、`internal/dep/git.go`、`internal/dep/bsr.go`、`internal/build/compiler.go` 关键节点 Info 级插桩（stage、kind、cache_hit、files、plugin、duration 属性）
  - [x] 6.1.4 验证测试通过（运行：`go test ./... -count=1`，输出干净；手动验证 `go run ./cmd/apigen entity list -f examples/book/api.yaml -v 2>/dev/null` stdout 无日志）
  - [x] 6.1.5 重构：统一日志属性 key 命名（常量或约定），消除散落字面量

- [x] 6.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试（`go test ./... -count=1`），确认输出干净后才继续
  - 调用 superpowers:requesting-code-review（占位符映射同 1.3，{BASE_SHA} 为本任务组开始前 SHA）
  - Critical/Important 处理流程同 1.3；Minor/无问题自动继续

## 7. PreCI 代码规范检查

<!-- 本项目不执行 PreCI 检查（用户明确要求），任务组 7 整体跳过 -->

- [x] 7.1 检测 preci 安装状态
  - 跳过：本项目不执行 PreCI
- [x] 7.2 检测项目是否已 preci 初始化
  - 跳过：本项目不执行 PreCI
- [x] 7.3 检测 PreCI Server 状态
  - 跳过：本项目不执行 PreCI
- [x] 7.4 执行代码规范扫描
  - 跳过：本项目不执行 PreCI
- [x] 7.5 处理扫描结果
  - 跳过：本项目不执行 PreCI

## 8. Documentation Sync (Required)

- [x] 8.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 8.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 8.3 sync proposal.md: update scope/impact if changed
- [x] 8.4 sync specs/*.md: update requirements if changed
- [x] 8.5 Final review: ensure all OpenSpec docs reflect actual implementation
