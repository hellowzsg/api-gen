## 1. YAML 配置扩展（create.key 解析与校验）

- [x] 1.1 create.key 配置解析与校验  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 1.1.1 写失败测试：`internal/yaml/parser_test.go`（解析 `create: { key: client }` / `{ key: server }` / `create: {}` 三种形态）、`internal/yaml/validate_test.go`（非法值如 `clinet` 报错，错误信息含实体名与合法取值）
  - [x] 1.1.2 验证测试失败（运行：`go test ./internal/yaml/ -count=1`，确认失败原因是 CreateDef/校验逻辑不存在）
  - [x] 1.1.3 写最小实现：`internal/yaml/parser.go`（新增 `CreateDef{ Key string }`，`Entity.Create` 类型 `*struct{}` → `*CreateDef`）、`internal/yaml/validate.go`（取值校验，空串等价 server）
  - [x] 1.1.4 验证测试通过（运行：`go test ./internal/yaml/... -count=1`，确认所有测试通过，输出干净）
  - [x] 1.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 1.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/create-client-key/specs/*.md` 和 `openspec/changes/create-client-key/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 2. Create client-key 代码生成（IR + 渲染）

- [x] 2.1 CreateIR 与渲染支持 client-key 模式  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 2.1.1 写失败测试：`internal/ir/builder_test.go`（client 模式 RequestFields 为 key=1 + 资源顺延 2..N+1、多资源顺延；server 模式不变；HTTP annotation 路径含 key 叶子段）、`internal/render/` 对应渲染测试（生成 proto 中 request 含 key 字段、HTTP path 含 `{key...}` 段）
  - [x] 2.1.2 验证测试失败（运行：`go test ./internal/ir/ ./internal/render/ -count=1`，确认失败原因是 client-key 逻辑不存在）
  - [x] 2.1.3 写最小实现：`internal/ir/builder.go`（`buildCreate` 按模式前置 key 字段；`buildCreateAnnotationWithResources` 在 client 模式拼 key 叶子路径）、`internal/render/template.go`（request message 与 HTTP path 渲染）
  - [x] 2.1.4 验证测试通过（运行：`go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 2.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 2.2 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/create-client-key/specs/*.md` 和 `openspec/changes/create-client-key/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 3. 端到端验证与文档

- [x] 3.1 examples/book 新增 client-key 实体 e2e 验证  <!-- TDD 任务：使用 5 步子任务 -->
  - [x] 3.1.1 写失败测试：`examples/book/e2e_http_test.go`（新增独立测试函数：POST `/library/<Service>/<entity>/{key...}` 创建成功，服务端收到的 key 由 path 段注入、资源由 body 注入；不改既有用例）
  - [x] 3.1.2 验证测试失败（运行：`cd examples/book && go test ./... -count=1`，确认失败原因是实体/路由不存在）
  - [x] 3.1.3 写最小实现：`examples/book/api.yaml`（新增一个 `create: { key: client }` 实体），重新生成 `examples/book/generated/`
  - [x] 3.1.4 验证测试通过（运行：`cd examples/book && go test ./... -count=1`，确认所有测试通过，输出干净）
  - [x] 3.1.5 重构：整理代码、改善命名、消除重复（保持所有测试通过）

- [x] 3.2 更新用户文档  <!-- 非 TDD 任务：使用 3 步子任务 -->
  - [x] 3.2.1 执行变更：`README.md`、`README_EN.md`（`create` 配置说明改为 server/client 两模式对照表）、`design-v2.md`（修订决策 #27：`create.key` 策略位落地；如新增 api-linter 豁免，同步 §16 豁免表）
  - [x] 3.2.2 验证无回归（运行：`go build ./... && go test ./... -count=1`，确认输出干净）
  - [x] 3.2.3 检查：确认文档描述与生成契约一致（请求形态、HTTP 路径、默认值），中英文档同步

- [x] 3.3 代码审查
  - 前置验证：调用 superpowers:verification-before-completion 运行全量测试，确认输出干净后才继续
  - 调用 superpowers:requesting-code-review 审查本任务组所有变更，占位符映射（以 OpenSpec 路径为准）：
    - `{PLAN_OR_REQUIREMENTS}` → `openspec/changes/create-client-key/specs/*.md` 和 `openspec/changes/create-client-key/tasks.md`
    - `{WHAT_WAS_IMPLEMENTED}` → 本任务组所有变更文件
    - `{BASE_SHA}` → 任务组开始前的 commit SHA（或分支基点）
    - `{HEAD_SHA}` → 当前 HEAD
  - 若存在 Critical/Important 问题：输出审查结果后追加选项提示，停止等待用户输入；用户选择"处理"类操作后，调用 superpowers:receiving-code-review 对每条审查意见做技术验证后再实施；按指令处理完成后继续下一任务组
  - 若仅有 Minor 或无问题：自动继续下一任务组，无需等待用户确认

## 4. Documentation Sync (Required)

- [x] 4.1 sync design.md: record technical decisions, deviations, and implementation details after each code change
- [x] 4.2 sync tasks.md: 逐一检查所有顶层任务及其子任务，将已完成但仍为 `[ ]` 的条目标记为 `[x]`；每次更新只修改 `[ ]` → `[x]`，禁止修改任何任务描述文字
- [x] 4.3 sync proposal.md: update scope/impact if changed
- [x] 4.4 sync specs/*.md: update requirements if changed
- [x] 4.5 Final review: ensure all OpenSpec docs reflect actual implementation
